package lib_overland

// autoupdate_version = 135

import (
	"context"
	"fmt"
	"os"

	client "github.com/influxdata/influxdb1-client/v2"

	// "go.mongodb.org/mongo-driver/bson"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const gps_log_entry_source_format = "Overland (%s)"
const gps_log_db_name = "ernie_org"
const gps_log_collection_name = "gps_log"

var mongodb_uri = os.Getenv("OVERLAND_MONGODB_URI")
var influxdb_uri = os.Getenv("OVERLAND_INFLUXDB_URI")

var global_mongodb_client *mongo.Client
var global_influxdb_client client.Client

func Write_location(ctx context.Context, l location) (*gps_log_point, error) {

	database := "ernie_org"
	measurement := "battery"

	gps_log_point, err := l.to_gps_log_point(fmt.Sprintf(gps_log_entry_source_format, l.Properties.Device_id))
	if err != nil {
		return nil, fmt.Errorf("Error converting location to gps_log_point: %w", err)
	}
	err = store_gps_point(ctx, gps_log_point)
	if err != nil {
		// just let it go
		log.Println("Got error storing gps point: %w", err)
	}

	log.Print(l)
	log.Println(l.Properties.Battery_level)
	log.Println(l.Properties.Device_id)
	log.Println(l.Properties.Battery_state)
	log.Println(l.Properties.Wifi)
	log.Println(l.Properties.Timestamp)
	log.Println("")
	influxdb_client, err := getInfluxClient()
	if err != nil {
		return nil, fmt.Errorf("Error retrieving new InfluxDB Client: %w", err)
	}

	// q := client.NewQuery("SELECT count(value) FROM cpu", "example", "")
	// if response, err := c.Query(q); err == nil && response.Error() == nil {
	// fmt.Println(response.Results)
	// }
	// pts =
	//pts := make([]client.Point, 1)

	tags := map[string]string{
		"wifi":          l.Properties.Wifi,
		"device_id":     l.Properties.Device_id,
		"battery_state": l.Properties.Battery_state,
	}
	fields := map[string]interface{}{
		"battery_level": l.Properties.Battery_level,
	}

	point_timestamp_str := l.Properties.Timestamp

	// 2019-10-29T20:01:11Z

	layout := "2006-01-02T15:04:05Z"

	point_timestamp_time, err := time.Parse(layout, point_timestamp_str)
	if err != nil {
		return nil, fmt.Errorf("error parsing time: %v: %w", point_timestamp_str, err)
	}

	pt, err := client.NewPoint(measurement, tags, fields, point_timestamp_time)
	if err != nil {
		return nil, fmt.Errorf("error creating influx point: %v/%v: %w", tags, fields, err)
	}
	// bps := client.BatchPoints{
	// Points:          pts,
	// Database:        database,
	// RetentionPolicy: "default",
	// }
	bp, _ := client.NewBatchPoints(client.BatchPointsConfig{
		Database: database,

		Precision: "s",
	})
	bp.AddPoint(pt)

	err = influxdb_client.Write(bp)
	if err != nil {
		return nil, fmt.Errorf("error writing point to influx: %v: %w", pt, err)
	}

	// measurements := client.BatchPoints{Points: pts, Database: database}
	// _, err = con.Write(measurements)

	return gps_log_point, nil
}

func store_gps_point(ctx context.Context, point *gps_log_point) error {

	log.Println("getting mongo collection object")

	mongodb_client, err := getMongoDBClient(ctx)
	if err != nil {
		wrappedErr := fmt.Errorf("got an error from getMongoDBClient(): %w", err)
		return wrappedErr
	}
	fmt.Println("got client:", mongodb_client)

	fmt.Println("connected client:", mongodb_client)
	collection := mongodb_client.Database(gps_log_db_name).Collection(gps_log_collection_name)

	log.Println("creating insert options")
	insert_opts := options.InsertOne()
	log.Println("inserting gps point")
	res, err := collection.InsertOne(ctx, point, insert_opts)
	if err != nil {
		return fmt.Errorf("got an error from InsertOne: %w", err)
	}
	log.Println("res: ", res)
	return nil
}
func (l *location) to_gps_log_point(entry_source string) (*gps_log_point, error) {
	log.Println("converting from location to gps_log_point")

	p := l.Properties

	log.Println("converting timestamp string to time.Time")
	point_timestamp_str := p.Timestamp

	// 2019-10-29T20:01:11Z

	layout := "2006-01-02T15:04:05Z"

	point_timestamp_time, err := time.Parse(layout, point_timestamp_str)
	if err != nil {
		return nil, fmt.Errorf("error parsing time: %v: %w", point_timestamp_str, err)
	}
	log.Println("looking for activity type")

	var activity_type string
	if len(p.Motion) > 0 {
		activity_type = p.Motion[0]
	}

	log.Println("returning new object")
	return &gps_log_point{
		Entry_source: entry_source,
		// ints
		Altitude:   float32(p.Altitude),
		Speed:      float32(p.Speed),
		Entry_date: point_timestamp_time,
		// lon, lat
		Loc: l.Geometry,
		// haven't seen this contain multiple values. Sometimes has zero.
		ActivityType: activity_type,
	}, nil
}

func MongoDBPing(ctx context.Context) (string, error) {

	mongodb_client, err := getMongoDBClient(ctx)
	if err != nil {
		return "", fmt.Errorf("Error getting MongoDB Client: %w", err)
	}
	err = mongodb_client.Ping(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("got an error from Ping(): %w", err)
	}
	return "OK", nil
}

func InfluxDBPing(ctx context.Context) (string, error) {
	influxdb_client, err := getInfluxClient()
	if err != nil {
		return "", fmt.Errorf("Error getting InfluxDB Client: %w", err)
	}

	dur, resp, err := influxdb_client.Ping(5 * time.Second)
	if err != nil {
		return "", fmt.Errorf("got an error from Ping(): %w", err)
	}
	return fmt.Sprintf("response: %v, duration: %v", resp, dur), nil
}

func getInfluxClient() (client.Client, error) {
	if influxdb_uri == "" {
		return nil, fmt.Errorf("no OVERLAND_INFLUXDB_URI env var set")
	}
	if global_influxdb_client != nil {
		log.Println("Re-using influx client")
		return global_influxdb_client, nil
	}
	var err error
	log.Println("Creating new influx client")
	global_influxdb_client, err = client.NewHTTPClient(client.HTTPConfig{
		Addr: influxdb_uri,
	})
	if err != nil {
		return nil, fmt.Errorf("Error creating new InfluxDB Client: %w", err)
	}
	return global_influxdb_client, nil
}
func getMongoDBClient(ctx context.Context) (*mongo.Client, error) {
	if mongodb_uri == "" {
		return nil, fmt.Errorf("no OVERLAND_MONGODB_URI env var set")
	}
	if global_mongodb_client != nil {
		log.Println("Re-using mongo client")
		return global_mongodb_client, nil
	}
	log.Println("Creating new mongo client")
	var err error
	global_mongodb_client, err = mongo.NewClient(options.Client().ApplyURI(mongodb_uri))
	if err != nil {
		return nil, fmt.Errorf("Error creating new MongoDB Client: %w", err)
	}
	global_mongodb_client.Connect(ctx)
	return global_mongodb_client, nil
}
