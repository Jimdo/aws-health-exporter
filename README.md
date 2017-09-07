# AWS Health Exporter [![Build Status](https://travis-ci.com/Jimdo/aws-health-exporter.svg?token=1djnvUyMgtcVefCz54T4&branch=master)](https://travis-ci.com/Jimdo/aws-health-exporter)

This is a simple server that scrapes the [AWS Status](https://status.aws.amazon.com/) (via the [AWS Health API](https://status.aws.amazon.com/)) and exports it via HTTP for Prometheus consumption. That allows you to alert on certain AWS status updates or to just make them visible on your dashboards.

_Note that in order to scrape the AWS Health API your AWS account has to have a Business or Enterprise support plan. See the [official documentation](http://docs.aws.amazon.com/health/latest/ug/what-is-aws-health.html) for details._

### Build
```
make
```

### Run
```
./aws-health-exporter --aws.region=eu-west-1
```

## Exposed metrics
The `aws-health-exporter` exports just one metric (event count) and you want to filter by the included labels.

Example
```
# This gives us a list of all `open` events in region `us-east-1` of type `issue`
aws_health_events{status_code="open", region="us-east-1", category="issue"}
```

Name | Description | Labels
-----|-----|-----
aws_health_events | AWS Health events | category, region, service, status_code

### Labels Explained
Label | Description
-----|-----
category | The category of the event. Possible events are issue, accountNotification and scheduledChange.
region | The AWS region name of the event. E.g. us-east-1.
service | The AWS service that is affected by the event. For example, EC2, RDS.
status_code | The most recent status of the event. Possible values are open, closed, and upcoming.

The labels match the corresponding `AWS Event` content - for a more detailed and up-to-date explanation see the offical documention [here](http://docs.aws.amazon.com/health/latest/APIReference/API_Event.html)

## Flags
Flag | Description
-----|-----
`--help` | Show help.
`--version` | Print version information
`--web.listen-address` | The address to listen on for HTTP requests. Default: ":9383"
`--aws.category` | A list of event type category codes (issue, scheduledChange, or accountNotification) that are used to filter events.
`--aws.region` | A list of AWS regions that are used to filter events
`--aws.service` | A list of AWS services that are used to filter events

## Docker
You can deploy this exporter using the [jimdo/aws-health-exporter](https://hub.docker.com/r/jimdo/aws-health-exporter/) Docker Image.

Example
```
docker pull jimdo/aws-health-exporter
docker run -p 9383:9383 jimdo/aws-health-exporter
```

### Credentials
The `aws-health-exporter` requires AWS credentials to access the AWS Health API. For example you can pass them via env vars using `-e AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY_ID} -e AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY}` options.

