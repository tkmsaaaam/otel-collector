package traceBufferRedis

import (
	"context"
	"encoding/json"
	"errors"
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
	"golang.org/x/exp/rand"
)

type traceBuffer struct {
	Context     context.Context
	RedisClient *redis.Client
	Marshaler   ptrace.JSONMarshaler
	Unmarshaler ptrace.JSONUnmarshaler
	Duration    time.Duration
	Consumer    consumer.Traces
	Traces      []*TraceMetadata
	Limit       int
	Rate        int
}

type TraceMetadata struct {
	Time time.Time       `json:"time"`
	Id   pcommon.TraceID `json:"id"`
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
		log.Println("can not make Metadata.(expired or incomplete)")
		return nil
	}

	i := rand.Intn(100)
	if i <= tb.Rate {
		log.Println("sampled TraceId: ", metadata.Id, ", time: ", metadata.Time)
		tb.Consumer.ConsumeTraces(ctx, td)
	}

	if metadata.Time.Before(now.Add(-tb.Duration)) {
		log.Println("consume expired TraceId: ", metadata.Id, ", time: ", metadata.Time)
		return nil
	}
	tb.Traces = push(tb.Traces, tb.Limit, metadata)
	bytes, e := tb.Marshaler.MarshalTraces(td)
	if e != nil {
		log.Println("err: ", e)
		return nil
	}
	err := tb.RedisClient.Set(context.Background(), makeKey(metadata.Id), string(bytes), tb.Duration).Err()
	if err != nil {
		log.Println("redis set err:", err)
		return nil
	}
	log.Println("cached TraceId: ", metadata.Id, ", time: ", metadata.Time)

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
		Context:     context,
		RedisClient: redisClient,
		Marshaler:   ptrace.JSONMarshaler{},
		Unmarshaler: ptrace.JSONUnmarshaler{},
		Duration:    d,
		Consumer:    consumer,
		Traces:      make([]*TraceMetadata, config.Limit),
		Limit:       config.Limit,
		Rate:        config.Rate,
	}
	go func() {
		http.HandleFunc("/flash", func(w http.ResponseWriter, r *http.Request) {
			flashHandler(w, r, tb)
		})
		log.Println("Server Start Up........")
		http.ListenAndServe(config.Host+":"+strconv.Itoa(config.Port), nil)
	}()
	return tb, nil
}

func flashHandler(w http.ResponseWriter, r *http.Request, tb *traceBuffer) {
	t := time.Now().Add(-tb.Duration)
	traces := []*TraceMetadata{}

	var start *time.Time = nil
	if r.FormValue("start") != "" {
		s, er := time.Parse(time.RFC3339, r.FormValue("start"))
		if er != nil {
			return
		}
		start = &s
	}
	var end *time.Time = nil
	if r.FormValue("end") != "" {
		e, er := time.Parse(time.RFC3339, r.FormValue("end"))
		if er != nil {
			return
		}
		end = &e
	}

	for _, v := range tb.Traces {
		if isContinue(v, start, end, t) {
			continue
		}
		res, err := tb.RedisClient.Get(context.Background(), makeKey(v.Id)).Result()
		if err != nil {
			log.Println("can not get trace Json: ", err)
			continue
		}
		trace, _ := tb.Unmarshaler.UnmarshalTraces([]byte(res))
		e := tb.Consumer.ConsumeTraces(context.Background(), trace)
		if e == nil {
			traces = append(traces, v)
		}
	}
	var res []byte
	var e error = nil
	res, e = json.Marshal(traces)
	if e != nil {
		log.Println("can not serialize", e)
		res = []byte("exported. can not serialize.")
	}
	tb.Traces = make([]*TraceMetadata, tb.Limit)
	_, err := w.Write(res)
	if err != nil {
		log.Println("can not write response:", err)
	} else {
		log.Println("flashed\n", string(res))
	}
}

func (t *TraceMetadata) MarshalJSON() ([]byte, error) {
	v, err := json.Marshal(&struct {
		Time time.Time
		Id   string
	}{
		Time: t.Time,
		Id:   t.Id.String(),
	})
	return v, err
}

func push(base []*TraceMetadata, limit int, meta *TraceMetadata) []*TraceMetadata {
	var start = 0
	len := len(base)
	if len >= limit {
		start = len - limit + 1
	}

	base = append(base, meta)
	sort.Slice(base, func(i, j int) bool {
		if base[j] == nil {
			return false
		}
		return base[i] == nil || base[i].Time.Before(base[j].Time)
	})
	return base[start:]
}

func makeKey(id pcommon.TraceID) string {
	return "trace:" + id.String()
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
	return &TraceMetadata{Id: id, Time: time}
}

func isContinue(v *TraceMetadata, start, end *time.Time, t time.Time) bool {
	if v == nil {
		return true
	}
	if v.Time.Before(t) {
		log.Println("expired TraceId: ", v.Id, ", time: ", v.Time)
		return true
	}
	if start != nil && start.After(v.Time) {
		return true
	}

	if end != nil && end.Before(v.Time) {
		return true
	}
	return false
}
