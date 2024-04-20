overlandreceiver: *.go */*.go test
	go build

.PHONY: deploy test

test:
	go test

deploy: overlandreceiver
	gcloud run deploy  --project=overland-receiver --source=. --env-vars-file=./ol.env  overlandreceiver

