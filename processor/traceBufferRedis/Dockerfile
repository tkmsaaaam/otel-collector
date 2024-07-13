# clamavreceiver:latest
FROM golang:1.22-bookworm as builder
WORKDIR /app
RUN git clone https://github.com/open-telemetry/opentelemetry-collector.git --depth=1
WORKDIR /app/opentelemetry-collector/cmd/mdatagen
RUN go install .
COPY *.go go.mod go.sum metadata.yaml /app/traceBufferRedis/
WORKDIR /app/traceBufferRedis
RUN go generate ./...
COPY otelcol-builder.yaml /app/otelcol-builder.yaml
WORKDIR /app
RUN go install go.opentelemetry.io/collector/cmd/builder@latest && builder --config=otelcol-builder.yaml

FROM debian:bookworm-slim
WORKDIR /app
COPY --from=builder /app/buffer /app/traceBufferRedis
COPY config.yaml /config/config.yaml
CMD ["/bin/bash", "-c", "/app/traceBufferRedis --config=/config/config.yaml"]
