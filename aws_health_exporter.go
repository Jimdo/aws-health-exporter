package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/health"
	"github.com/aws/aws-sdk-go/service/health/healthiface"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/version"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	// APIRegion is the region where the AWS health api lives. Currently this only `us-east-1`.
	APIRegion = "us-east-1"

	// Namespace..
	namespace = "aws_health"
)

var (
	labels = []string{
		"availability_zone",
		"event_type_category",
		"region",
		"service",
	}

	counters = map[string]*prometheus.CounterVec{
		"open": prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "open",
			Help:      "Counter for open events",
		}, labels),
		"upcoming": prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "upcoming",
			Help:      "Counter for upcoming events",
		}, labels),
		"closed": prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "closed",
			Help:      "Counter for closed events",
		}, labels),
	}
)

type exporter struct {
	api    healthiface.HealthAPI
	filter *health.EventFilter
	m      sync.Mutex
}

func (e *exporter) Describe(ch chan<- *prometheus.Desc) {
	for _, c := range counters {
		c.Describe(ch)
	}
}

func (e *exporter) Collect(ch chan<- prometheus.Metric) {
	e.m.Lock()
	defer e.m.Unlock()

	events := query(e.api, e.filter)

	for _, c := range counters {
		c.Reset()
	}

	for _, e := range events {
		c := counters[*e.StatusCode]
		c.WithLabelValues(
			aws.StringValue(e.AvailabilityZone),
			aws.StringValue(e.EventTypeCategory),
			aws.StringValue(e.Region),
			aws.StringValue(e.Service)).Inc()
	}

	for _, c := range counters {
		c.Collect(ch)
	}
}

func query(api healthiface.HealthAPI, filter *health.EventFilter) (events []*health.Event) {
	err := api.DescribeEventsPages(&health.DescribeEventsInput{
		Filter:     filter,
		MaxResults: aws.Int64(10),
	}, func(out *health.DescribeEventsOutput, lastPage bool) bool {
		events = append(events, out.Events...)
		return true
	})
	if err != nil {
		log.Println(err)
		return
	}
	return
}

func init() {
	prometheus.MustRegister(version.NewCollector("aws_health_exporter"))
}

func main() {
	var (
		showVersion       = kingpin.Flag("version", "Print version information").Bool()
		listenAddr        = kingpin.Flag("web.listen-address", "The address to listen on for HTTP requests.").Default(":9383").String()
		availabilityZones = kingpin.Flag("aws.availability-zone", "A list of AWS availability zones.").Strings()
		categories        = kingpin.Flag("aws.event-type-category", "A list of event type category codes (issue, scheduledChange, or accountNotification).").Strings()
		regions           = kingpin.Flag("aws.region", "A list of AWS regions.").Strings()
		services          = kingpin.Flag("aws.service", "A list of AWS services.").Strings()
	)

	registerSignals()

	kingpin.Parse()

	if *showVersion {
		fmt.Fprintln(os.Stdout, version.Print("aws_health_exporter"))
		os.Exit(0)
	}

	sess, err := session.NewSession(&aws.Config{Region: aws.String(APIRegion)})
	if err != nil {
		log.Fatal(err)
	}

	filter := &health.EventFilter{}
	if len(*availabilityZones) > 0 {
		filter.AvailabilityZones = aws.StringSlice(*availabilityZones)
	}
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
	mux.Handle("/metrics", prometheus.Handler())
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
