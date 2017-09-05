FROM golang:alpine

ARG VERSION
ARG DATE

ADD . /go/src/github.com/Jimdo/aws-health-exporter
WORKDIR /go/src/github.com/Jimdo/aws-health-exporter

RUN go install -ldflags="-X 'main.Version=${VERSION}' -X 'main.BuildTime=${DATE}'" ./...

ENTRYPOINT  [ "/go/bin/aws-health-exporter" ]
EXPOSE      9383
