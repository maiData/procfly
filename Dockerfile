FROM nats:2.9 as nats

FROM natsio/prometheus-nats-exporter:0.10.1 as nats-metrics

FROM golang:1.19 as procfly

WORKDIR /build

COPY . /build

RUN go install .

FROM busybox:1.35-glibc as busybox

FROM gcr.io/distroless/base

COPY --from=nats /nats-server /bin/nats-server
COPY --from=nats-metrics /prometheus-nats-exporter /bin/prometheus-nats-exporter

WORKDIR /
ENTRYPOINT [ "procfly" ]

COPY --from=procfly /go/bin/procfly /bin/procfly

COPY example /
