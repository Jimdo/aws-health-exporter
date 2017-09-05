package main

import (
	"reflect"
	"sort"
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
	ch := make(chan prometheus.Metric)

	go func() {
		defer close(ch)
		e.Collect(ch)
	}()

	validateMetric(t, ch, events[0], 1.)
	validateMetric(t, ch, events[1], 1.)
	validateMetric(t, ch, events[2], 3.)
}

func validateMetric(t *testing.T, ch <-chan prometheus.Metric, e *health.Event, expectedVal float64) {
	m := <-ch
	pb := &dto.Metric{}
	m.Write(pb)

	labels := pb.GetLabel()
	expectedLabels := getLabelsFromEvent(e)
	sort.Sort(prometheus.LabelPairSorter(labels))
	sort.Sort(prometheus.LabelPairSorter(expectedLabels))

	if !reflect.DeepEqual(labels, expectedLabels) {
		t.Errorf("Invalid labels - Expected: %v Got: %v", expectedLabels, labels)
	}

	val := pb.GetGauge().GetValue()
	if pb.GetGauge().GetValue() != expectedVal {
		t.Errorf("Invalid value - Expected: %v Got: %v", expectedVal, val)
	}
}

func getLabelsFromEvent(e *health.Event) []*dto.LabelPair {
	return []*dto.LabelPair{
		&dto.LabelPair{
			Name:  aws.String(LabelEventTypeCategory),
			Value: e.EventTypeCategory,
		},
		&dto.LabelPair{
			Name:  aws.String(LabelRegion),
			Value: e.Region,
		},
		&dto.LabelPair{
			Name:  aws.String(LabelService),
			Value: e.Service,
		},
		&dto.LabelPair{
			Name:  aws.String(LabelStatusCode),
			Value: e.StatusCode,
		},
	}
}
