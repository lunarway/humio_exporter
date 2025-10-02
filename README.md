# humio_exporter

[![Build Status](https://travis-ci.com/lunarway/humio_exporter.svg?branch=master)](https://travis-ci.com/lunarway/humio_exporter)
[![Go Report Card](https://goreportcard.com/badge/github.com/lunarway/humio_exporter)](https://goreportcard.com/report/github.com/lunarway/humio_exporter)
[![Docker Repository on Quay](https://quay.io/repository/lunarway/humio_exporter/status "Docker Repository on Quay")](https://quay.io/repository/lunarway/humio_exporter)

Prometheus exporter for [Humio](https://humio.com/) written in Go.

# Installation

Several pre-compiled binaries are available from the [releases page](https://github.com/lunarway/humio_exporter/releases).

A docker image is also available on our Quay.io registry.

```
docker run quay.io/lunarway/humio_exporter --humio.api-token <api-token> --humio.url <humio-api-url> --config queries.yaml
```

# Usage

You need an API Token with access to the repositories you specify in the configuration file. You can find you API Token in the "Your Account" in the top right corner of your Humio installation.

The exporter exposes prometheus metrics on `/metrics` on port `9534` (can be configured).

To specify which queries you want exported as Prometheus metrics, you have to provide a configuration file, e.g. `queries.yaml` in the format:

```
queries:
- query: count(status)
  repo: humio
  interval: 30m
  metric_name: humio_status_total

- query: max(status)
  repo: humio
  interval: 30m
  metric_name: humio_status_max

- query: min(status)
  repo: humio
  interval: 30m
  metric_name: humio_status_min

- query: avg(status)
  repo: humio
  interval: 30m
  metric_name: humio_status_avg

- query: counterAsRate(status)
  repo: humio
  interval: 30m
  metric_name: humio_status_rate

- query: range(status)
  repo: humio
  interval: 30m
  metric_name: humio_status_range

- query: stdDev(field=status)
  repo: humio
  interval: 30m
  metric_name: humio_status_stddev

- query: sum(status)
  repo: humio
  interval: 30m
  metric_name: humio_status_sum
  metric_labels:
  - key: squad
    value: foo
```

As seen in the last example query, you can also specify a set of static labels to be outputtet along with the metric.

## Table-based Metrics with Dynamic Labels

The exporter also supports table-based queries that return multiple rows, where each row can be exported as a separate metric with dynamic labels derived from table columns. This is useful for queries using `groupBy()` or similar functions that return tabular data.

Example configuration for table-based metrics:

```yaml
- query: eventSize() | groupBy("k8s.labels.app", function=sum(_eventSize, as="value"))
  repo: humio
  interval: 1h
  metric_name: log_ingest_bytes
  metric_labels:
  - key: app
    valueFromTable: k8s.labels.app
```

This configuration will:
1. Execute a query that groups events by `k8s.labels.app` and sums the event sizes
2. Return a table with columns like `k8s.labels.app` and `value`
3. Create separate metrics for each row, using the `k8s.labels.app` column value as the `app` label
4. Result in metrics like:
   - `log_ingest_bytes{app="foo", interval="1h", repo="humio"} 100`
   - `log_ingest_bytes{app="bar", interval="1h", repo="humio"} 200`

The `valueFromTable` field specifies which table column should be used as the label value. The metric value is automatically extracted from common field names like `value`, `_value`, `count`, or `_count`.

**Null Value Handling**: If a table column contains null values or empty strings, the corresponding label will be set to "unknown". This ensures consistent metric cardinality and follows Prometheus best practices. You can see this behavior in debug logs.

Currently the export supports the above aggregate query functions and table-based queries with dynamic labels.

```
humio_exporter --config=CONFIG --humio.url=HUMIO.URL --humio.api-token=API.TOKEN
```

See all configuration options with the `--help` flag

```
$ humio_exporter --help
usage: humio_exporter --config=CONFIG --humio.url=HUMIO.URL --humio.api-token=API.TOKEN [<flags>]

Humio exporter for Prometheus. Provide your Humio API token and configuration file with queries to expose as Prometheus metrics.

Flags:
  -h, --help                 Show context-sensitive help (also try --help-long and --help-man).
      --config=CONFIG        The humio_exporter configuration file to be used
      --humio.url=HUMIO.URL  Humio base API url
      --humio.api-token=API.TOKEN  Humio API token
      --humio.timeout=10     Timeout for requests against the Humio API
      --web.listen-address=":9534"
                             Address on which to expose metrics.
      --log.level="info"     Only log messages with the given severity or above. Valid levels: [debug, info, warn, error, fatal]
      --log.format="logger:stderr"
                             Set the log target and format. Example: "logger:syslog?appname=bob&local=7" or
                             "logger:stdout?json=true"
      --version              Show application version.
```

It is possible to use a file to pass arguments to the exporter.
For example:

```
 echo --humio.api-token=<>\n > args
```

And run the exporter using:

```
./humio_exporter @args
```

# Design

The specified queries in the configuration file will be exporter with two labels:

- `repo` - the repository that the query was executed against
- `interval` - the interval the query result represent.

There is an option to add static labels as well. This can be done as follows:

```
- query: sum(status)
  repo: humio
  interval: 30m
  metric_name: humio_status_sum
  metric_labels:
  - key: squad
    value: foo
```

Example.

```
humio_total{interval="5m", repo="humio"} 3458
humio_audit_total{interval="5m", repo="humio-audit"} 2976
humio_status_sum{interval="30m", repo="humio", squad="foo"} 235
```

# Build

The exporter can be build using the standard Go tool chain if you have it available.

```
go build
```

You can build inside a docker image as well.
This produces a `humio_exporter` image that can run with the binary as entry point.

```
docker build -t humio_exporter .
```

This is useful if the exporter is to be depoyled in Kubernetes or other dockerized environments.

# Development

The project uses Go modules so you need Go version >=1.11 to run it.
Run builds and tests with the standard Go tool chain.

```go
go build
go test
```

# Deployment

To deploy the exporter in Kubernetes, you can find a simple Kubernetes deployment and secret yaml in the `examples` folder. You have to add your Humio api token in the `secrets.yaml` and/or the url of you humio deployment `deployment.yaml`. The examples assumes that you have a namespace in kubernetes named: `monitoring`.

It further assumes that you have [kubernetes service discovery](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#kubernetes_sd_config) configured for you Prometheus instance and a target that will gather metrics from pods, similar to this:

```
- job_name: 'kubernetes-pods'
  kubernetes_sd_configs:
  - role: pod

  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    action: keep
    regex: true
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    action: replace
    target_label: __metrics_path__
    regex: (.+)
  - source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
    action: replace
    regex: (.+):(?:\d+);(\d+)
    replacement: ${1}:${2}
    target_label: __address__
  - action: labelmap
    regex: __meta_kubernetes_pod_label_(.+)
```

To deploy it to your kubernetes cluster run the following commands:

```
kubectl create configmap humio-exporter-config --from-file=examples/queries.yaml --namespace=monitoring
kubectl apply -f examples/secrets.yaml
kubectl apply -f examples/deployment.yaml
```

The exporter expose http endpoints that can be used by kubernetes probes:

- `/healthz` - used for liveness probe, always returns `healthy`, status code 200.
- `/ready` - used for readiness probe, return `true` and status code 200 after the first scrape completed. Otherwise, it returns `false`, with status code 503.
