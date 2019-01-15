# humio_exporter

[![Build Status](https://travis-ci.com/lunarway/humio_exporter.svg?branch=master)](https://travis-ci.com/lunarway/humio_exporter)
[![Go Report Card](https://goreportcard.com/badge/github.com/lunarway/humio_exporter)](https://goreportcard.com/report/github.com/lunarway/humio_exporter)
[![Docker Repository on Quay](https://quay.io/repository/lunarway/humio_exporter/status "Docker Repository on Quay")](https://quay.io/repository/lunarway/humio_exporter)

Prometheus exporter for [Humio](https://humio.com/) written in Go.

# Installation

Several pre-compiled binaries are available from the [releases page](https://github.com/lunarway/humio_exporter/releases).

A docker image is also available on our Quay.io registry.

```
docker run quay.io/lunarway/humio_exporter --api.token <api-token> --humio.url <humio-api-url> --config queries.yaml
```

# Usage

You need an API Token with access to the repositories you specify in the configuration file. You can find you API Token in the "Your Account" in the top right corner of your Humio installation.

The exporter exposes prometheus metrics on `/metrics` on port `9534` (can be configured).

To specify which queries you want exported as Prometheus metrics, you have to provide a configuration file, e.g. `queries.yaml` in the format: 

```
queries:
- query: count()
  repo: humio-audit
  interval: 5m
  metric_name: log_lines

- query: count()
  repo: humio
  interval: 5m
  metric_name: some_metric
```
NB! Currently the exporter only supports `count()` queries.

```
humio_exporter --config=CONFIG --humio.url=HUMIO.URL --api.token=API.TOKEN
```

See all configuration options with the `--help` flag

```
$ humio_exporter --help
usage: humio_exporter --config=CONFIG --humio.url=HUMIO.URL --api.token=API.TOKEN [<flags>]

Humio exporter for Prometheus. Provide your Humio API token and configuration file with queries to expose as Prometheus metrics.

Flags:
  -h, --help                 Show context-sensitive help (also try --help-long and --help-man).
      --config=CONFIG        The humio_exporter configuration file to be used
      --humio.url=HUMIO.URL  Humio base API url
      --api.token=API.TOKEN  Humio API token
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
 echo --api.token=<>\n > args
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