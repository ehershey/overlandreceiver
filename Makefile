overlandreceiver: *.go */*.go test
	go build

.PHONY: deploy test

test:
	go test

deploy: overlandreceiver
	gcloud run deploy  --project=overland-receiver --source=. overlandreceiver

curl:
	curl --data ' { "locations": [ { "type": "Feature", "geometry": { "type": "Point", "coordinates": [ -122.030581, 37.331800 ] }, "properties": { "timestamp": "2015-10-01T08:00:00Z", "altitude": 0 } }]} ' https://$host/
