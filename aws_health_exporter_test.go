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

func TestCollect(t *testing.T) {
	var events = []*health.Event{
		&health.Event{
			AvailabilityZone:  aws.String(""),
			EventTypeCategory: aws.String("issue"),
			Region:            aws.String("eu-west-1"),
			Service:           aws.String("EC2"),
			StatusCode:        aws.String("closed"),
		},
		&health.Event{
			AvailabilityZone:  aws.String(""),
			EventTypeCategory: aws.String("issue"),
			Region:            aws.String("eu-west-1"),
			Service:           aws.String("EC2"),
			StatusCode:        aws.String("closed"),
		},
	}
	e := &exporter{
		api: &mockHealthAPI{events: events},
		filter: &health.EventFilter{
			Regions: aws.StringSlice([]string{"us-west-1"}),
		},
	}
	ch := make(chan prometheus.Metric)

	go func() {
		defer close(ch)
		e.Collect(ch)
	}()

	if expect, got := 2., readCounter((<-ch).(prometheus.Counter)); expect != got {
		t.Errorf("Counter 'closed' has wrong value. Expected: %v Got: %v", expect, got)
	}
}

func TestFilter(t *testing.T) {
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
			Region:            aws.String("us-west-1"),
			Service:           aws.String("LAMBDA"),
			StatusCode:        aws.String("open"),
		},
	}
	e := &exporter{
		api: &mockHealthAPI{events: events},
		filter: &health.EventFilter{
			Regions: aws.StringSlice([]string{"us-west-1"}),
		},
	}
	ch := make(chan prometheus.Metric)

	go func() {
		defer close(ch)
		e.Collect(ch)
	}()

	if expect, got := 1., readCounter((<-ch).(prometheus.Counter)); expect != got {
		t.Errorf("Counter 'open' has wrong value. Expected: %v Got: %v", expect, got)
	}
}

func readCounter(m prometheus.Metric) float64 {
	pb := &dto.Metric{}
	m.Write(pb)
	return pb.GetCounter().GetValue()
}

var mockEvents = []*health.Event{
	&health.Event{
		AvailabilityZone:  aws.String(""),
		EventTypeCategory: aws.String("issue"),
		Region:            aws.String("eu-west-1"),
		Service:           aws.String("EC2"),
		StatusCode:        aws.String("closed"),
	},
	&health.Event{
		AvailabilityZone:  aws.String(""),
		EventTypeCategory: aws.String("issue"),
		Region:            aws.String("us-east-1"),
		Service:           aws.String("RDS"),
		StatusCode:        aws.String("open"),
	},
	&health.Event{
		AvailabilityZone:  aws.String(""),
		EventTypeCategory: aws.String("issue"),
		Region:            aws.String("us-east-1"),
		Service:           aws.String("RDS"),
		StatusCode:        aws.String("closed"),
	},
	&health.Event{
		AvailabilityZone:  aws.String(""),
		EventTypeCategory: aws.String("issue"),
		Region:            aws.String("us-east-1"),
		Service:           aws.String("RDS"),
		StatusCode:        aws.String("closed"),
	},
	&health.Event{
		AvailabilityZone:  aws.String(""),
		EventTypeCategory: aws.String("issue"),
		Region:            aws.String("us-west-1"),
		Service:           aws.String("LAMBDA"),
		StatusCode:        aws.String("upcoming"),
	},
}
