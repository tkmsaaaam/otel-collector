package traceBufferRedis

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
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
}

// Capabilities implements processor.Traces.
func (t *traceBuffer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// ConsumeTraces implements processor.Traces.
func (tb *traceBuffer) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	b, e := tb.marshaler.MarshalTraces(td)
	if e != nil {
		log.Println("err: ", e)
	}
	key := "trace:" + strconv.FormatInt(time.Now().Unix(), 10)
	err := tb.redisClient.Set(context.Background(), key, string(b), tb.duration).Err()
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
	d, _ := time.ParseDuration(config.expire)
	redisClient := redis.NewClient(&redis.Options{
		Addr: config.redisUrl,
		DB:   config.dbName,
	})
	tb := &traceBuffer{
		context:     context,
		redisClient: redisClient,
		marshaler:   ptrace.JSONMarshaler{},
		unmarshaler: ptrace.JSONUnmarshaler{},
		duration:    d,
		consumer:    consumer,
	}
	go func() {
		http.HandleFunc("/flash", func(w http.ResponseWriter, r *http.Request) {
			flashHandler(w, r, tb)
		})
		log.Println("Server Start Up........")
		http.ListenAndServe("localhost:"+strconv.Itoa(config.port), nil)
	}()
	return tb, nil
}

func flashHandler(w http.ResponseWriter, _ *http.Request, tb *traceBuffer) {
	rdb := tb.redisClient
	res := rdb.Keys(context.Background(), "trace:*")
	keys, _ := res.Result()
	for _, key := range keys {
		res, err := rdb.Get(context.Background(), key).Bytes()
		if err != nil {
			log.Println(err)
		}
		trace, _ := tb.unmarshaler.UnmarshalTraces(res)
		t, _ := strconv.ParseInt(strings.Replace(key, "trace:", "", 1), 10, 64)
		log.Println(time.Unix(t, 0))
		tb.consumer.ConsumeTraces(context.Background(), trace)
	}

	hello := []byte("Hello World!!!")
	_, err := w.Write(hello)
	if err != nil {
		log.Fatal(err)
	}
}
