FROM golang:1.6-alpine

MAINTAINER Alessandro Arnone <arnone.alessandro@gmail.com>

ADD . /go/src/github.com/aarnone/httpbatcher

# The binary is created in the cmd/httpbatcher package
RUN apk add --no-cache git mercurial
RUN go get -d github.com/aarnone/httpbatcher/cmd/httpbatcher 
RUN apk del git mercurial
RUN go install github.com/aarnone/httpbatcher/cmd/httpbatcher

ENTRYPOINT /go/bin/httpbatcher

EXPOSE 8080
