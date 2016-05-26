FROM debian:jessie

MAINTAINER Alessandro Arnone <arnone.alessandro@gmail.com>

RUN apt-get update
RUN apt-get install -y ca-certificates

COPY ./httpbatcher /httpbatcher

EXPOSE 8080

CMD ["/httpbatcher"]
