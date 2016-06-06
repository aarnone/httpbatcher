FROM golang:1.6-alpine

MAINTAINER Alessandro Arnone <arnone.alessandro@gmail.com>

ADD . /go/src/github.com/aarnone/httpbatcher

# The binary is created in the cmd/httpbatcher package
RUN go install github.com/aarnone/httpbatcher/cmd/httpbatcher

ENTRYPOINT /go/bin/httpbatcher

EXPOSE 8080
