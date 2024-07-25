package traceBufferRedis

import (
	"testing"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestPush(t *testing.T) {
	now := time.Now()
	type Param struct {
		base  []*TraceMetadata
		limit int
		meta  *TraceMetadata
	}
	tests := []struct {
		name  string
		param Param
		want  []*TraceMetadata
	}{
		{
			name:  "param size 1, limit 1",
			param: Param{base: []*TraceMetadata{&TraceMetadata{Id: pcommon.NewTraceIDEmpty(), Time: now}}, limit: 1, meta: &TraceMetadata{Id: pcommon.NewTraceIDEmpty(), Time: now}},
			want:  []*TraceMetadata{&TraceMetadata{Id: pcommon.NewTraceIDEmpty(), Time: now}},
		},
		{
			name:  "param size 1(nil), limit 1",
			param: Param{base: []*TraceMetadata{nil}, limit: 1, meta: &TraceMetadata{Id: pcommon.NewTraceIDEmpty(), Time: now}},
			want:  []*TraceMetadata{&TraceMetadata{Id: pcommon.NewTraceIDEmpty(), Time: now}},
		},
		{
			name:  "param size 2, limit 1",
			param: Param{base: []*TraceMetadata{&TraceMetadata{Id: pcommon.NewTraceIDEmpty(), Time: now}, &TraceMetadata{Id: pcommon.NewTraceIDEmpty(), Time: now}}, limit: 1, meta: &TraceMetadata{Id: pcommon.NewTraceIDEmpty(), Time: now}},
			want:  []*TraceMetadata{&TraceMetadata{Id: pcommon.NewTraceIDEmpty(), Time: now}},
		},
		{
			name:  "param size 1, limit 2",
			param: Param{base: []*TraceMetadata{&TraceMetadata{Id: pcommon.NewTraceIDEmpty(), Time: now}}, limit: 2, meta: &TraceMetadata{Id: pcommon.NewTraceIDEmpty(), Time: now}},
			want:  []*TraceMetadata{&TraceMetadata{Id: pcommon.NewTraceIDEmpty(), Time: now}, &TraceMetadata{Id: pcommon.NewTraceIDEmpty(), Time: now}},
		},
		{
			name:  "param size 1, limit 3",
			param: Param{base: []*TraceMetadata{&TraceMetadata{Id: pcommon.NewTraceIDEmpty(), Time: now}}, limit: 3, meta: &TraceMetadata{Id: pcommon.NewTraceIDEmpty(), Time: now}},
			want:  []*TraceMetadata{&TraceMetadata{Id: pcommon.NewTraceIDEmpty(), Time: now}, &TraceMetadata{Id: pcommon.NewTraceIDEmpty(), Time: now}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			actual := push(tt.param.base, tt.param.limit, tt.param.meta)

			if len(tt.want) != len(actual) {
				t.Errorf("push() length actual: \n%v\nwant: \n%v", len(actual), len(tt.want))
			}
			for i := 0; i < len(tt.want); i++ {
				w := tt.want[i]
				a := actual[i]
				if w.Id.String() != a.Id.String() {
					t.Errorf("push() %v id actual: \n%v\nwant: \n%v", i, a.Id, w.Id)
				}
				if w.Time.String() != a.Time.String() {
					t.Errorf("push() %v actual: \n%v\nwant: \n%v", i, a.Time, w.Time)
				}
			}
		})
	}
}

func TestMakeTraceMetadata(t *testing.T) {
	tests := []struct {
		name        string
		setFunction func(ptrace.Traces) ptrace.Traces
		want        *TraceMetadata
	}{
		{
			name:        "future",
			setFunction: makeFutureSpan,
			want:        nil,
		},
		{
			name:        "present",
			setFunction: makeSpan,
			want:        &TraceMetadata{Id: pcommon.NewTraceIDEmpty(), Time: time.Date(2024, 6, 26, 3, 14, 45, 10, &time.Location{})},
		},
		{
			name:        "oldest one",
			setFunction: makeMultipleSpans,
			want:        &TraceMetadata{Id: pcommon.NewTraceIDEmpty(), Time: time.Date(2024, 6, 26, 3, 14, 45, 9, &time.Location{})},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			now := time.Date(2024, 6, 26, 3, 14, 45, 11, &time.Location{})
			input := tt.setFunction(ptrace.NewTraces())

			actual := makeTraceMetaData(input, now)
			if tt.want == nil {
				if tt.want != actual {
					t.Errorf("makeTraceMetaData() actual: \n%v\nwant: \n%v", actual, tt.want)
				}
			} else {
				if tt.want.Id.String() != actual.Id.String() {
					t.Errorf("makeTraceMetaData() id: actual: \n%v\nwant: \n%v", actual, tt.want)
				}
				if tt.want.Time.String() != actual.Time.String() {
					t.Errorf("makeTraceMetaData() time: actual: \n%v\nwant: \n%v", actual, tt.want)
				}
			}
		})
	}
}

func makeFutureSpan(base ptrace.Traces) ptrace.Traces {
	start := time.Date(2024, 6, 26, 3, 14, 45, 11, &time.Location{})
	end := time.Date(2024, 6, 26, 3, 14, 45, 12, &time.Location{})
	span := base.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.SetEndTimestamp(pcommon.NewTimestampFromTime(end))
	span.SetStartTimestamp(pcommon.NewTimestampFromTime(start))
	return base
}
func makeSpan(base ptrace.Traces) ptrace.Traces {
	start := time.Date(2024, 6, 26, 3, 14, 45, 10, &time.Location{})
	end := time.Date(2024, 6, 26, 3, 14, 45, 12, &time.Location{})
	span := base.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.SetTraceID(pcommon.NewTraceIDEmpty())
	span.SetEndTimestamp(pcommon.NewTimestampFromTime(end))
	span.SetStartTimestamp(pcommon.NewTimestampFromTime(start))
	return base
}

func makeMultipleSpans(base ptrace.Traces) ptrace.Traces {

	resourceSpans := base.ResourceSpans()

	span := resourceSpans.AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.SetTraceID(pcommon.NewTraceIDEmpty())
	span.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Date(2024, 6, 26, 3, 14, 45, 10, &time.Location{})))
	span.SetEndTimestamp(pcommon.NewTimestampFromTime(time.Date(2024, 6, 26, 3, 14, 45, 12, &time.Location{})))

	span1 := resourceSpans.AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span1.SetTraceID(pcommon.NewTraceIDEmpty())
	span1.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Date(2024, 6, 26, 3, 14, 45, 9, &time.Location{})))
	span1.SetEndTimestamp(pcommon.NewTimestampFromTime(time.Date(2024, 6, 26, 3, 14, 45, 12, &time.Location{})))
	return base
}

func TestIsContinue(t *testing.T) {
	type Param struct {
		v     *TraceMetadata
		start time.Time
		end   time.Time
		t     time.Time
	}
	tests := []struct {
		name  string
		param Param
		want  bool
	}{
		{
			name:  "v is nil",
			param: Param{v: nil, start: time.Date(2024, 7, 10, 0, 0, 0, 0, &time.Location{}), end: time.Date(2024, 7, 11, 0, 0, 0, 0, &time.Location{}), t: time.Date(2024, 7, 10, 12, 0, 0, 0, &time.Location{})},
			want:  true,
		},
		{
			name:  "v.Time  is before",
			param: Param{v: &TraceMetadata{Id: pcommon.NewTraceIDEmpty(), Time: time.Date(2024, 7, 10, 11, 59, 59, 59, &time.Location{})}, start: time.Date(2024, 7, 10, 0, 0, 0, 0, &time.Location{}), end: time.Date(2024, 7, 11, 0, 0, 0, 0, &time.Location{}), t: time.Date(2024, 7, 10, 12, 0, 0, 0, &time.Location{})},
			want:  true,
		},
		{
			name:  "start  is before",
			param: Param{v: &TraceMetadata{Id: pcommon.NewTraceIDEmpty(), Time: time.Date(2024, 7, 9, 23, 59, 59, 59, &time.Location{})}, start: time.Date(2024, 7, 10, 0, 0, 0, 0, &time.Location{}), end: time.Date(2024, 7, 11, 0, 0, 0, 0, &time.Location{}), t: time.Date(2024, 7, 10, 12, 0, 0, 0, &time.Location{})},
			want:  true,
		},
		{
			name:  "end  is after",
			param: Param{v: &TraceMetadata{Id: pcommon.NewTraceIDEmpty(), Time: time.Date(2024, 7, 11, 0, 0, 0, 1, &time.Location{})}, start: time.Date(2024, 7, 10, 0, 0, 0, 0, &time.Location{}), end: time.Date(2024, 7, 11, 0, 0, 0, 0, &time.Location{}), t: time.Date(2024, 7, 10, 12, 0, 0, 0, &time.Location{})},
			want:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			actual := isContinue(tt.param.v,&tt.param.start, &tt.param.end, tt.param.t)

			if tt.want != actual {
				t.Errorf("isContinue() actual: \n%v\nwant: \n%v", actual, tt.want)
			}
		})
	}
}
