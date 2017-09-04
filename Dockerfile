FROM golang:alpine

ADD . /go/src/github.com/Jimdo/aws-health-exporter
WORKDIR /go/src/github.com/Jimdo/aws-health-exporter

RUN go install -v ./...

ENTRYPOINT  [ "/go/bin/aws-health-exporter" ]
EXPOSE      9231
