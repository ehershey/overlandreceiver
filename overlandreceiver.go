package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	"github.com/ehershey/overlandreceiver/lib_overland"
	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
)

const FilenameTemplate = "/%s/posts.txt"
const Port = 8080

const autoupdate_version = 48

var request_timeout time.Duration // incoming requests
const request_timeout_seconds = 9

const min_foot_location_count = 7 // how many points of walking or running must be seen to re-generate activity data

var battery string = ""

func filename() string {
	var home string
	if runtime.GOOS == "linux" {
		home = "home"
	} else {
		home = "Users"
	}
	return fmt.Sprintf(FilenameTemplate, home)
}

func getHandler(w http.ResponseWriter, r *http.Request) {

	log.Println("spitting out status and battery")
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "{ \"result\": \"ok\", \"battery\": \"%v\"}\n", battery)
}

func handler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), request_timeout)
	defer cancel()
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub().Clone()
		ctx = sentry.SetHubOnContext(ctx, hub)
	}

	options := []sentry.SpanOption{
		sentry.WithOpName("http.server"),
		sentry.ContinueFromRequest(r),
		sentry.WithTransactionSource(sentry.SourceURL),
	}

	transaction := sentry.StartTransaction(ctx,
		fmt.Sprintf("%s %s", r.Method, r.URL.Path),
		options...,
	)
	defer transaction.Finish()

	var post lib_overland.Overlandpost

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&post)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Println(err)
		return
	} else {
		if decoder.More() {
			http.Error(w, "Trailing garbage in body", http.StatusBadRequest)
			log.Println("Trailing garbage in body")
			return
		} else {
			log.Println(post)
			f, err := os.OpenFile(filename(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				log.Println(err)
				return
			}
			json, err := json.Marshal(post)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				log.Println(err)
				return
			}

			saw_foot_location_count := 0
			// battery = fmt.Sprintf("%v", json)
			for _, location := range post.Locations {
				gps_point, err := lib_overland.Write_location(location)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					log.Println(err)
					return
				}
				if gps_point.ActivityType == "running" || gps_point.ActivityType == "walking" {
					saw_foot_location_count += 1
				}

				// fmt.Println(location.Properties.Device_id)
				// fmt.Println(location.Properties.Battery_level)
				// fmt.Println(location.Properties.Battery_state)
				// fmt.Println(location.Properties.Wifi)
				// fmt.Println("")
				battery = fmt.Sprintf("%.2f", location.Properties.Battery_level)
			}
			if _, err := f.Write(json); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				log.Println(err)
				return
			}
			if _, err := f.Write([]byte("\n")); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				log.Println(err)
				return
			}
			if err := f.Close(); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				log.Println(err)
				return
			}
			log.Println("Wrote to file.")
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, "{\"name\": \"E-Locations\", \"result\": \"ok\"}")
			log.Printf("Checking whether to update day data: %d > %d?\n", saw_foot_location_count, min_foot_location_count)
			if saw_foot_location_count > min_foot_location_count {
				go UpdateDayData()
			}
		}
	}
}

func main() {
	request_timeout = time.Duration(request_timeout_seconds * time.Second)
	err := sentry.Init(sentry.ClientOptions{
		Debug:              false,
		EnableTracing:      true,
		TracesSampleRate:   1.0,
		ProfilesSampleRate: 1.0,
		AttachStacktrace:   true,
	})
	if err != nil {
		log.Fatalf("sentry.Init: %s", err)
	}
	defer sentry.Flush(2 * time.Second)

	f, err := os.OpenFile(filename(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	if err := f.Close(); err != nil {
		panic(err)
	}

	sentryHandler := sentryhttp.New(sentryhttp.Options{})

	http.HandleFunc("/", sentryHandler.HandleFunc(handler))

	http.HandleFunc("/overland", sentryHandler.HandleFunc(getHandler))

	// http.HandleFunc("/", handler)
	// http.HandleFunc("/overland", getHandler)
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(Port))
	if err != nil {
		panic(err)
	}
	log.Println("Listening on port", Port)
	log.Println("autoupdate_version", autoupdate_version)

	log.Fatal(http.Serve(listener, nil))
}

// [ ] read post data
// [ ] append it to a file
// [ ] test with curl
// [ ] test with big value

func UpdateDayData() {
	//
	// last_overland_update=db.getSiblingDB("ernie_org").daily_summary.find().sort({"latest_timestamp_by_entry_source.Overland":-1}).limit(1).toArray()[0].latest_timestamp_by_entry_source.Overland
	datestring := time.Now().Format("2006-01-02")
	log.Println("starting call sudo")
	home, err := os.UserHomeDir()
	if err != nil {
		wrappedErr := fmt.Errorf("Error getting my homedir: %v", err)
		log.Println("got an error:", wrappedErr)
		return
	}
	command_argv0 := fmt.Sprintf("%s/new_oura_activity.sh", home)
	cmd := exec.Command(command_argv0, datestring)
	cmd.Env = os.Environ()
	log.Println("cmd:", cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Println(err)
	} else {
		log.Println("> new_oura_activity.sh:")
		log.Println(string(out))
	}
	log.Println("ending call sudo")
}
