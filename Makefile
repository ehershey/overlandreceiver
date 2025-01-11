bearer=$(shell grep OVERLAND_REQUIRED_BEARER temp.env | cut -f2 -d=)

MODULE_VERSION = $(shell git describe --tags)
GOFLAGS = -tags BuildArgsIncluded

LDFLAGS = -X \"main.ModuleVersion=$(MODULE_VERSION)\"


overlandreceiver: *.go */*.go test
	go build -ldflags "$(LDFLAGS)" $(GOFLAGS)

docker-build: overlandreceiver start.sh
	docker build -t overlandreceiver .

docker-run: docker-build temp.env
	docker run --env-file=./temp.env --network=bridge -p 0:8080 overlandreceiver

temp.env:
	@echo Create temp.env manually

.PHONY: deploy test docker-build

test:
	go test $(GOFLAGS)

deploy: overlandreceiver
	gcloud run deploy  --project=overland-receiver --source=. overlandreceiver --region=us-east1

curl:
	curl --header "Authorization: Bearer $(bearer)" https://$(host)/version
	curl --header "Authorization: Bearer $(bearer)" https://$(host)/mongodbhealth
	curl --header "Authorization: Bearer $(bearer)" https://$(host)/influxdbhealth
	curl --header "Authorization: Bearer $(bearer)" --data ' { "locations": [ { "type": "Feature", "geometry": { "type": "Point", "coordinates": [ -122.030581, 37.331800 ] }, "properties": { "timestamp": "2015-10-01T08:00:00Z", "altitude": 0 } }]} ' https://$(host)/
