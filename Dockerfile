FROM golang:1.6-alpine

MAINTAINER Alessandro Arnone <arnone.alessandro@gmail.com>

# Install git, needed for go get command
RUN apk add --no-cache git

ADD . /go/src/github.com/aarnone/httpbatcher

# The binary is created in the cmd/httpbatcher package
RUN go get -d github.com/aarnone/httpbatcher/cmd/httpbatcher
RUN go install github.com/aarnone/httpbatcher/cmd/httpbatcher

# Free some space
RUN apk del git

ENTRYPOINT /go/bin/httpbatcher

EXPOSE 8080
