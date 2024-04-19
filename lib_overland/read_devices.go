package lib_overland

// autoupdate_version = 85

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/davecgh/go-spew/spew"
	client "github.com/influxdata/influxdb1-client/v2"
)

func Read_devices() []Device {
	if influxdb_uri == "" {
		log.Fatalf("no OVERLAND_INFLUXDB_URI env var set")
	}

	database := "ernie_org"
	measurement := "battery"

	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr: influxdb_uri,
	})
	if err != nil {
		fmt.Println("Error creating InfluxDB Client: ", err.Error())
	}
	defer c.Close()

	sql := fmt.Sprintf("select * from %v where time > now() - 10m group by device_id order by time desc limit 1", measurement)

	q := client.NewQuery(sql, database, "")
	response, err := c.Query(q)
	if err != nil {
		panic(err)
	}
	if response.Error() != nil {
		panic(response.Error())
	}

	var devices []Device

	for index, result := range response.Results {
		fmt.Println("index: ", index)
		fmt.Println("result: ", result)
		// fmt.Println("%#v", result) // will print the data in go-syntax form.
		// fmt.Println("%T", result)  // will print the data's type in go-syntax form.
		fmt.Println(spew.Sdump(result))

		for _, row := range result.Series {
			device := Device{Name: row.Tags["device_id"]}
			for column_index, column := range row.Columns {
				value := row.Values[0][column_index]
				if column == "battery_level" {
					percentage, err := value.(json.Number).Float64()
					if err != nil {
						panic(err)
					}
					device.Percentage = percentage * 100
				} else if column == "battery_state" {
					device.State = value.(string)
				} else if column == "wifi" && value != nil {
					device.Wifi = value.(string)
				} else if column == "time" {
					device.Timestamp = value.(string)
				}
			}

			devices = append(devices, device)
		}
	}

	// {
	// [
	// {battery map[device_id:eahxs] [time battery_level battery_state wifi] [[2019-11-13T16:25:04Z 1 charging <nil>]] false}
	// // {battery map[device_id:eahipad] [time battery_level battery_state wifi] [[2019-11-13T16:22:55Z 0.9700000286102295 charging <nil>]] false }
	// ] []
	// }

	return devices
	// []Device{{"eahxs", .98, "Charging", " 🥰", "homeblahblah", "2019-12-12 12:11:11.00", "17s"}}
}
