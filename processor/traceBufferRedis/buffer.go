package traceBufferRedis

import (
	"context"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
)

type traceBuffer struct {
	context     context.Context
	redisClient *redis.Client
	marshaler   ptrace.JSONMarshaler
	unmarshaler ptrace.JSONUnmarshaler
	duration    time.Duration
	consumer    consumer.Traces
	traces      []*TraceMetadata
	limit       int
}

type TraceMetadata struct {
	time time.Time
	id   pcommon.TraceID
}

// Capabilities implements processor.Traces.
func (t *traceBuffer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// ConsumeTraces implements processor.Traces.
func (tb *traceBuffer) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	now := time.Now()
	metadata := makeTraceMetaData(td, now)
	if metadata == nil {
		return nil
	}
	if metadata.time.Before(now.Add(-tb.duration)) {
		return nil
	}
	tb.traces = push(tb.traces, tb.limit, metadata)
	bytes, e := tb.marshaler.MarshalTraces(td)
	if e != nil {
		log.Println("err: ", e)
	}
	err := tb.redisClient.Set(context.Background(), makeKey(metadata.id.String()), string(bytes), tb.duration).Err()
	if err != nil {
		log.Println("redis set err:", err)
	}
	// TODO: sampling
	return nil
}

// Shutdown implements processor.Traces.
func (t *traceBuffer) Shutdown(ctx context.Context) error {
	return nil
}

// Start implements processor.Traces.
func (t *traceBuffer) Start(ctx context.Context, host component.Host) error {
	return nil
}

func newTraceBuffer(context context.Context, config *Config, consumer consumer.Traces) (processor.Traces, error) {
	d, _ := time.ParseDuration(config.Expire)
	redisClient := redis.NewClient(&redis.Options{
		Addr: config.RedisUrl,
		DB:   config.DbName,
	})
	tb := &traceBuffer{
		context:     context,
		redisClient: redisClient,
		marshaler:   ptrace.JSONMarshaler{},
		unmarshaler: ptrace.JSONUnmarshaler{},
		duration:    d,
		consumer:    consumer,
		traces:      make([]*TraceMetadata, config.Limit),
		limit:       config.Limit,
	}
	go func() {
		http.HandleFunc("/flash", func(w http.ResponseWriter, r *http.Request) {
			flashHandler(w, r, tb)
		})
		log.Println("Server Start Up........")
		http.ListenAndServe("localhost:"+strconv.Itoa(config.Port), nil)
	}()
	return tb, nil
}

func flashHandler(w http.ResponseWriter, _ *http.Request, tb *traceBuffer) {
	t := time.Now().Add(-tb.duration)
	for _, v := range tb.traces {
		if t.Before(v.time) {
			res, err := tb.redisClient.Get(context.Background(), makeKey(v.id.String())).Result()
			if err != nil {
				log.Println(err)
			}
			trace, _ := tb.unmarshaler.UnmarshalTraces([]byte(res))
			tb.consumer.ConsumeTraces(context.Background(), trace)
		}
	}
	hello := []byte("Hello World!!!")
	_, err := w.Write(hello)
	if err != nil {
		log.Fatal(err)
	}
}

func push(base []*TraceMetadata, limit int, meta *TraceMetadata) []*TraceMetadata {
	var start = 0
	len := len(base)
	if len > limit {
		start = len - limit
	}

	base = append(base, meta)
	sort.Slice(base, func(i, j int) bool {
		return base[i].time.Before(base[j].time)
	})
	return base[start:]
}

func makeKey(id string) string {
	return "trace:" + id
}

func makeTraceMetaData(td ptrace.Traces, time time.Time) *TraceMetadata {
	var b bool = false
	var id pcommon.TraceID
	for i := 0; i < td.ResourceSpans().Len(); i++ {
		recourceSpan := td.ResourceSpans().At(i)
		for j := 0; j < recourceSpan.ScopeSpans().Len(); j++ {
			scopeSpan := recourceSpan.ScopeSpans().At(j)
			for k := 0; k < scopeSpan.Spans().Len(); k++ {
				span := scopeSpan.Spans().At(k)
				if span.StartTimestamp().AsTime().Before(time) {
					b = true
					id = span.TraceID()
					time = span.StartTimestamp().AsTime()
				}
			}
		}
	}
	if !b {
		return nil
	}
	return &TraceMetadata{id: id, time: time}
}
