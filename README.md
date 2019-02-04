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
    value: nasa
```
As seen in the last example query, you can also specify a set of static labels to be outputtet along with the metric.

Currently the export supports the above aggregate query functions

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

Here is an example.

```
humio_total{interval="5m", repo="humio"} 3458
humio_audit_total{interval="5m", repo="humio-audit"} 2976
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
* `/healthz` - used for liveness probe, always returns `healthy`, status code 200.
* `/ready` - used for readiness probe, return `true` and status code 200 after the first scrape completed. Otherwise, it returns `false`, with status code 503.
