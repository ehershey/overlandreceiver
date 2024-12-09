package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/ehershey/overlandreceiver/lib_overland"
	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
)

const Port = 8080

const autoupdate_version = 110

var general_timeout time.Duration // used for everything
const general_timeout_seconds = 10

const min_foot_location_count = 7 // how many points of walking or running must be seen to re-generate activity data

var battery string = ""

var requiredBearer = os.Getenv("OVERLAND_REQUIRED_BEARER")

func getHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("spitting out status and battery")
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "{ \"result\": \"ok\", \"battery\": \"%v\"}\n", battery)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), general_timeout)
	defer cancel()
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		fmt.Printf("creating new hub\n")
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
	ctx = transaction.Context()

	var post lib_overland.Overlandpost

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&post)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Println(err)
		return
	}
	if decoder.More() {
		http.Error(w, "Trailing garbage in body", http.StatusBadRequest)
		log.Println("Trailing garbage in body")
		return
	}
	log.Printf("post: %v\n", post)

	saw_foot_location_count := 0
	// battery = fmt.Sprintf("%v", json)
	for _, location := range post.Locations {
		gps_point, err := lib_overland.Write_location(ctx, location)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
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
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(w, "{\"name\": \"E-Locations\", \"result\": \"ok\"}")
	log.Printf("Checking whether to update day data: %d > %d?\n", saw_foot_location_count, min_foot_location_count)
	if saw_foot_location_count > min_foot_location_count {
		go UpdateDayData()
	}
}

func main() {
	general_timeout = time.Duration(general_timeout_seconds * time.Second)
	err := sentry.Init(sentry.ClientOptions{
		Debug:              true,
		EnableTracing:      true,
		TracesSampleRate:   1.0,
		ProfilesSampleRate: 1.0,
		AttachStacktrace:   true,
	})
	if err != nil {
		log.Fatalf("sentry.Init: %s", err)
	}
	defer sentry.Flush(2 * time.Second)
	sentry.CaptureMessage("my message")

	sentryHandler := sentryhttp.New(sentryhttp.Options{})

	if err := preflight(); err != nil {
		log.Fatalf("Error running preflight checks: %v", err)
	}

	http.HandleFunc("/", ernieHandleFunc(sentryHandler.HandleFunc(rootHandler)))

	http.HandleFunc("/version", ernieHandleFunc(sentryHandler.HandleFunc(versionHandler)))
	http.HandleFunc("/mongodbhealth", ernieHandleFunc(sentryHandler.HandleFunc(mongoDBhealthHandler)))
	http.HandleFunc("/influxdbhealth", ernieHandleFunc(sentryHandler.HandleFunc(influxDBhealthHandler)))

	http.HandleFunc("/overland", ernieHandleFunc(sentryHandler.HandleFunc(getHandler)))

	listener, err := net.Listen("tcp", ":"+strconv.Itoa(Port))
	if err != nil {
		panic(err)
	}
	log.Println("Listening on port", Port)
	log.Println("autoupdate_version", autoupdate_version)

	log.Fatal(http.Serve(listener, nil))
}

func UpdateDayData() {
	datestring := time.Now().Format("2006-01-02")
	log.Println("starting call sudo")
	home, err := os.UserHomeDir()
	if err != nil {
		wrappedErr := fmt.Errorf("Error getting my homedir: %v", err)
		log.Println("got an error:", wrappedErr)
		return
	}
	command_argv0 := fmt.Sprintf("echo %s/new_oura_activity.sh", home)
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

func versionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "{\"version\":\"%v\"}\n", autoupdate_version)
}

func influxDBhealthHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), general_timeout)
	defer cancel()
	resp, err := lib_overland.InfluxDBPing(ctx)
	if err != nil {
		wrappedErr := fmt.Errorf("Error pinging InfluxDB: %w", err)
		log.Println("got an error:", wrappedErr)
		http.Error(w, wrappedErr.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "{\"health\":\"%v\"}\n", resp)
}

func mongoDBhealthHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), general_timeout)
	defer cancel()
	resp, err := lib_overland.MongoDBPing(ctx)
	if err != nil {
		wrappedErr := fmt.Errorf("Error pinging MongoDB: %w", err)
		log.Println("got an error:", wrappedErr)
		http.Error(w, wrappedErr.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "{\"health\":\"%v\"}\n", resp)
}
func preflight() error {
	var errs error
	log.Printf("Performing preflight checks\n")
	log.Printf("InfluxDB Ping:\n")
	ctx, cancel := context.WithTimeout(context.Background(), general_timeout)
	defer cancel()
	resp, err := lib_overland.InfluxDBPing(ctx)
	if err != nil {
		errs = err
	}
	log.Printf("%v\n", resp)
	log.Printf("MongoDB Ping:\n")
	ctx, cancel = context.WithTimeout(context.Background(), general_timeout)
	defer cancel()
	resp, err = lib_overland.MongoDBPing(ctx)
	if err != nil {
		if errs != nil {
			errs = errors.Join(errs, err)
		} else {
			errs = err
		}
	}

	if requiredBearer == "" {
		errs = errors.Join(errs, fmt.Errorf("Missing environment variable: OVERLAND_REQUIRED_BEARER"))
	}

	log.Printf("%v\n", resp)
	return errs
}

func ernieHandleFunc(handler http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("got request (%s)!\n", r.URL)
		passedAuth := r.Header.Get("Authorization")
		if passedAuth == "" {
			http.Error(w, "", 401)
			return
		}
		requiredAuth := fmt.Sprintf("Bearer %s", requiredBearer)
		if passedAuth != requiredAuth {
			http.Error(w, "", 403)
			return
		}
		handler(w, r)
	})
}
