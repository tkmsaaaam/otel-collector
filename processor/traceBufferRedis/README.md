# Trace Buffer Redis

This processor caches Traces. Specifically it sends to Redis. It sends to exporter when the specified  endpoint (default: `localhost:8080/flash`) is called.

## What author thinks

Current tracing and sampling schemes require that the sampling rate be pre-set. (There are exceptions, such as Tail Sampling Processor.)

However, the need for tracing is not always constant. For example, the need for tracing before and after a failure is very high. On the other hand, the need for tracing is relatively low during normal times. (Of course, it is important to have high observability even during normal times.)

Therefore, in order to always obtain a satisfactory trace, it is necessary to set the sampling rate in advance according to the highest sampling rate. In this case, however, the cost of the tracing system can be an issue. This is because many commercial tracing systems are pay-as-you-go based on the number of spans.

To solve the problem of sending spans to a tracing system that charges excessively and has the restriction that the sampling rate must be set in advance, we created a processor that caches traces in Redis under normal conditions and outputs them as needed.

In this way, traces are sent to the tracing system at the appropriate sampling rate during normal times, and when a higher sampling rate is required, such as during a failure, the cached traces are sent by calling a pre-defined endpoint.

## Options

- `expire`: Traces are output to the tracing system from the time the endpoint is called back to that time period. (default: `1m`)
- `redis_url`: `host:port` (default: `localhost:6379`)
- `db_name`: Redis db name. (default: `0`)
- `port`: Port of the endpoint to call when outputting to the tracing system.`localhost:port/flash`(default: `8080`)
- `limit`: Maximum number of ***traces*** output to the tracing system when the endpoint is called. (default: `1000`)

## Examples

```yml
processors:
  traceBufferRedis:
    expire: 1m
    redis_url: redis:6379
    db_name: 0
    host: "0.0.0.0"
    port: 8080
    limit: 1000
    rate: 50
```

## Author

tkmsaaaam
