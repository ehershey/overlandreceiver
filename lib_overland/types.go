package lib_overland

import (
	"time"
)

type geometry struct {
	Coordinates []float64 `json:"coordinates"`
	Type        string    `json:"type"`
}

type properties struct {
	Activity                        string   `json:"activity"`
	Altitude                        int      `json:"altitude"`
	Battery_level                   float64  `json:"battery_level"`
	Battery_level_of_something_else float64  `json:"battery_level_of_something_else"`
	Battery_state                   string   `json:"battery_state"`
	Deferred                        float64  `json:"deferred"`
	Desired_accuracy                int      `json:"desired_accuracy"`
	Device_id                       string   `json:"device_id"`
	Horizontal_accuracy             int      `json:"horizontal_accuracy"`
	Locations_in_payload            int      `json:"locations_in_payload"`
	Motion                          []string `json:"motion"`
	Pauses                          bool     `json:"pauses"`
	Significant_change              int      `json:"significant_change"`
	Speed                           int      `json:"speed"`
	Timestamp                       string   `json:"timestamp"`
	Vertical_accuracy               int      `json:"vertical_accuracy"`
	Wifi                            string   `json:"wifi"`
}

type location struct {
	Geometry   geometry   `json:"geometry"`
	Properties properties `json:"properties"`
	Type       string     `json:"type"`
}

type Overlandpost struct {
	Locations []location `json:"locations"`
}

type Device struct {
	Name       string
	Percentage float64
	State      string
	StateIcon  string
	Wifi       string
	Timestamp  string
	Age        string
}

/*
		{
	    _id: ObjectId("6000859d66805292d8094e3e"),
	    altitude: 4.45,
	    speed: 2.46,
	    entry_source: 'Garmin Livetrack',
	    entry_date: 2021-01-13T21:27:09.000Z,
	    loc: {
	      type: 'Point',
	      coordinates: [ -73.93383446149528, 40.62351515516639 ]
	    },
	    activityType: 'RUNNING'
	  }
*/
type gps_log_point struct {
	Entry_source string    `json:"entry_source"`
	Altitude     float32   `json:"altitude"`
	Speed        float32   `json:"speed"`
	Entry_date   time.Time `json:"entry_date"`
	Loc          geometry  `json:"loc"`
	ActivityType string    `json:"activityType"`
	Elevation    float32   `json:"elevation,omitempty"`
}
