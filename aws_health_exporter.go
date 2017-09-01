package main

import (
	"flag"
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
)

const (
	// APIRegion is the region where the AWS health api lives. Currently this only `us-east-1`.
	APIRegion = "us-east-1"

	// Namespace..
	namespace = "aws_health"
)

var (
	labels = []string{
		"AvailabilityZone",
		"EventTypeCategory",
		"EventTypeCode",
		"Region",
		"Service",
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
	api healthiface.HealthAPI
	m   sync.Mutex
}

func (e *exporter) Describe(ch chan<- *prometheus.Desc) {
	for _, c := range counters {
		c.Describe(ch)
	}
}

func (e *exporter) Collect(ch chan<- prometheus.Metric) {
	e.m.Lock()
	defer e.m.Unlock()

	events := query(e.api)

	for _, c := range counters {
		c.Reset()
	}

	for _, e := range events {
		c := counters[*e.StatusCode]
		c.WithLabelValues(
			aws.StringValue(e.AvailabilityZone),
			aws.StringValue(e.EventTypeCategory),
			aws.StringValue(e.EventTypeCode),
			aws.StringValue(e.Region),
			aws.StringValue(e.Service)).Inc()
	}

	for _, c := range counters {
		c.Collect(ch)
	}
}

func query(api healthiface.HealthAPI) (events []*health.Event) {
	err := api.DescribeEventsPages(&health.DescribeEventsInput{
		Filter: &health.EventFilter{
		// Regions: []*string{aws.String("eu-west-1")}, // todo: expose filters
		},
		MaxResults: aws.Int64(10),
	}, func(out *health.DescribeEventsOutput, lastPage bool) bool {
		for _, e := range out.Events {
			events = append(events, e)
		}
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
		listenAddr  = flag.String("web.listen-address", ":9229", "The address to listen on for HTTP requests.")
		showVersion = flag.Bool("version", false, "Print version information")
	)

	registerSignals()

	flag.Parse()

	if *showVersion {
		fmt.Fprintln(os.Stdout, version.Print("aws_health_exporter"))
		os.Exit(0)
	}

	sess, err := session.NewSession(&aws.Config{Region: aws.String(APIRegion)})
	if err != nil {
		log.Fatal(err)
	}
	exporter := &exporter{api: health.New(sess)}
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
