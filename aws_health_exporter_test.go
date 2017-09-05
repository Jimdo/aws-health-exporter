package main

import (
	"log"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/health"
	"github.com/aws/aws-sdk-go/service/health/healthiface"
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
			AvailabilityZone:  aws.String(""),
			EventTypeCategory: aws.String("issue"),
			Region:            aws.String("eu-west-1"),
			Service:           aws.String("EC2"),
			StatusCode:        aws.String("open"),
		},
		&health.Event{
			AvailabilityZone:  aws.String(""),
			EventTypeCategory: aws.String("issue"),
			Region:            aws.String("us-east-1"),
			Service:           aws.String("EC2"),
			StatusCode:        aws.String("open"),
		},
		&health.Event{
			AvailabilityZone:  aws.String(""),
			EventTypeCategory: aws.String("issue"),
			Region:            aws.String("us-east-1"),
			Service:           aws.String("LAMBDA"),
			StatusCode:        aws.String("closed"),
		},
		&health.Event{
			AvailabilityZone:  aws.String(""),
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

	e.scrape()

	if expect, got := 2., readCounter("", "issue", "us-east-1", "LAMBDA", "closed"); expect != got {
		t.Errorf("Counter 'closed' has wrong value. Expected: %v Got: %v", expect, got)
	}
	if expect, got := 1., readCounter("", "issue", "us-east-1", "EC2", "open"); expect != got {
		t.Errorf("Counter 'open' has wrong value. Expected: %v Got: %v", expect, got)
	}
}

func readCounter(availabilityZone, eventTypeCategory, region, service, statusCode string) float64 {
	vec, ok := counters[statusCode]
	if !ok {
		log.Fatalf("no CounterVec for status code %v", statusCode)
	}
	m := vec.WithLabelValues(availabilityZone, eventTypeCategory, region, service)
	pb := &dto.Metric{}
	m.Write(pb)
	return pb.GetCounter().GetValue()
}
