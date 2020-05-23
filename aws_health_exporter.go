package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"text/tabwriter"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/health"
	"github.com/aws/aws-sdk-go/service/health/healthiface"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	// APIRegion is the region where the AWS health api lives. Currently
	// this is only `us-east-1`, see: http://docs.aws.amazon.com/health/latest/ug/getting-started-api.html
	APIRegion = "us-east-1"

	// LabelCategory defines the event type category of the event, e.g. issue, accountNotification, scheduledChange
	LabelCategory = "category"
	// LabelRegion defines the region of the event, e.g. us-east-1
	LabelRegion = "region"
	// LabelService defines the service of the event, e.g. EC2, RDS
	LabelService = "service"
	// LabelStatusCode defines the status of the event, e.g. open, upcoming, closed
	LabelStatusCode = "status_code"
	// Namespace is the metrics prefix
	Namespace = "aws_health"
)

var (
	// BuildTime represents the time of the build
	BuildTime = "N/A"
	// Version represents the Build SHA-1 of the binary
	Version = "N/A"

	// labels are the static labels that come with every metric
	labels = []string{LabelCategory, LabelRegion, LabelService, LabelStatusCode}

	// events is the number of aws health events reported
	eventOpts = prometheus.GaugeOpts{
		Name:      "events",
		Namespace: Namespace,
		Help:      "Gauge for aws health events",
	}
)

type exporter struct {
	api    healthiface.HealthAPI
	filter *health.EventFilter
}

func (e *exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- prometheus.NewDesc(
		prometheus.BuildFQName(eventOpts.Namespace, eventOpts.Subsystem, eventOpts.Name),
		eventOpts.Help,
		labels,
		nil,
	)
}

func (e *exporter) Collect(ch chan<- prometheus.Metric) {
	gv := prometheus.NewGaugeVec(eventOpts, labels)
	e.scrape(gv)
	gv.Collect(ch)
}

func (e *exporter) scrape(gv *prometheus.GaugeVec) {
	var events []*health.Event

	err := e.api.DescribeEventsPages(&health.DescribeEventsInput{
		Filter: e.filter,
	}, func(out *health.DescribeEventsOutput, lastPage bool) bool {
		events = append(events, out.Events...)
		return true
	})

	if err != nil {
		log.Println(err)
		return
	}

	for _, e := range events {
		gv.WithLabelValues(
			aws.StringValue(e.EventTypeCategory),
			aws.StringValue(e.Region),
			aws.StringValue(e.Service),
			aws.StringValue(e.StatusCode)).Inc()
	}
}

func init() {
	prometheus.MustRegister(version.NewCollector("aws_health_exporter"))
}

func main() {
	var (
		showVersion = kingpin.Flag("version", "Print version information").Bool()
		listenAddr  = kingpin.Flag("web.listen-address", "The address to listen on for HTTP requests.").Default(":9383").String()
		categories  = kingpin.Flag("aws.category", "A list of event type category codes (issue, scheduledChange, or accountNotification) that are used to filter events.").Strings()
		regions     = kingpin.Flag("aws.region", "A list of AWS regions that are used to filter events").Strings()
		services    = kingpin.Flag("aws.service", "A list of AWS services that are used to filter events").Strings()
	)

	registerSignals()

	kingpin.Parse()

	if *showVersion {
		tw := tabwriter.NewWriter(os.Stdout, 2, 1, 2, ' ', 0)
		fmt.Fprintf(tw, "Build Time:   %s\n", BuildTime)
		fmt.Fprintf(tw, "Build SHA-1:  %s\n", Version)
		fmt.Fprintf(tw, "Go Version:   %s\n", runtime.Version())
		tw.Flush()
		os.Exit(0)
	}

	log.Printf("Starting `aws-health-exporter`: Build Time: '%s' Build SHA-1: '%s'\n", BuildTime, Version)

	sess, err := session.NewSession(&aws.Config{Region: aws.String(APIRegion)})
	if err != nil {
		log.Fatal(err)
	}

	filter := &health.EventFilter{}
	if len(*categories) > 0 {
		filter.EventTypeCategories = aws.StringSlice(*categories)
	}
	if len(*regions) > 0 {
		filter.Regions = aws.StringSlice(*regions)
	}
	if len(*services) > 0 {
		filter.Services = aws.StringSlice(*services)
	}

	exporter := &exporter{api: health.New(sess), filter: filter}
	prometheus.MustRegister(exporter)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>AWS Health Exporter</title></head>
             <body>
             <h1>AWS Health Exporter</h1>
             <p><a href='/metrics'>Metrics</a></p>
             </body>
             </html>`))
	})
	log.Println("Listening on", *listenAddr)
	http.ListenAndServe(*listenAddr, mux)
}

func registerSignals() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Print("Received SIGTERM, exiting...")
		os.Exit(1)
	}()
}
