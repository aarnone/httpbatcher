# httpbatcher

[![Build Status](https://travis-ci.org/aarnone/httpbatcher.svg?branch=master)](https://travis-ci.org/aarnone/httpbatcher)

A proxy that process multiple http request packed in a single one. Similar to Google batch API and OData.

# Build instruction

To build the project just run:
```
$ go test
$ go build
```

To run the integration test suite you'll need docker and docker-compose available, and run:
```
$ GOOS=linux go build ./cmd/httpbatcher
$ docker-compose up -d --build
$ go test ./integration-test --tags integration
$ docker-compose down -v --rmi local
```
