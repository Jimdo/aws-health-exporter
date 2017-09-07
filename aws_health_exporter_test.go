package main

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/health"
	"github.com/aws/aws-sdk-go/service/health/healthiface"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

type mockHealthAPI struct {
	healthiface.HealthAPI
	events []*health.Event
}

func (api *mockHealthAPI) DescribeEventsPages(in *health.DescribeEventsInput, fn func(*health.DescribeEventsOutput, bool) bool) error {
	output := health.DescribeEventsOutput{Events: api.events}
	fn(&output, false)
	return nil
}

func TestScrape(t *testing.T) {
	var events = []*health.Event{
		&health.Event{
			EventTypeCategory: aws.String("issue"),
			Region:            aws.String("eu-west-1"),
			Service:           aws.String("EC2"),
			StatusCode:        aws.String("open"),
		},
		&health.Event{
			EventTypeCategory: aws.String("issue"),
			Region:            aws.String("us-east-1"),
			Service:           aws.String("EC2"),
			StatusCode:        aws.String("open"),
		},
		&health.Event{
			EventTypeCategory: aws.String("issue"),
			Region:            aws.String("us-east-1"),
			Service:           aws.String("LAMBDA"),
			StatusCode:        aws.String("closed"),
		},
		&health.Event{
			EventTypeCategory: aws.String("issue"),
			Region:            aws.String("us-east-1"),
			Service:           aws.String("LAMBDA"),
			StatusCode:        aws.String("closed"),
		},
		&health.Event{
			EventTypeCategory: aws.String("issue"),
			Region:            aws.String("us-east-1"),
			Service:           aws.String("LAMBDA"),
			StatusCode:        aws.String("closed"),
		},
	}
	e := &exporter{
		api:    &mockHealthAPI{events: events},
		filter: &health.EventFilter{},
	}

	gv := prometheus.NewGaugeVec(eventOpts, labels)
	e.scrape(gv)

	validateMetric(t, gv, events[0], 1.)
	validateMetric(t, gv, events[1], 1.)
	validateMetric(t, gv, events[2], 3.)
}

func validateMetric(t *testing.T, vec *prometheus.GaugeVec, e *health.Event, expectedVal float64) {
	m := vec.WithLabelValues(*e.EventTypeCategory, *e.Region, *e.Service, *e.StatusCode)
	pb := &dto.Metric{}
	m.Write(pb)

	val := pb.GetGauge().GetValue()
	if pb.GetGauge().GetValue() != expectedVal {
		t.Errorf("Invalid value - Expected: %v Got: %v", expectedVal, val)
	}
}
