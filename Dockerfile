FROM golang:1.18.2 as builder
WORKDIR /src
ENV CGO_ENABLED=0
ENV GOOS=linux

RUN apt-get update
RUN apt-get install -y ca-certificates

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go build -o /tmp/humio_exporter

FROM scratch

ENTRYPOINT [ "/humio_exporter" ]
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /tmp/humio_exporter /
COPY examples/queries.yaml /
