package traceBufferRedis

import (
	"testing"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

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
			want:        &TraceMetadata{id: pcommon.NewTraceIDEmpty(), time: time.Date(2024, 6, 26, 3, 14, 45, 10, &time.Location{})},
		},
		{
			name:        "oldest one",
			setFunction: makeMultipleSpans,
			want:        &TraceMetadata{id: pcommon.NewTraceIDEmpty(), time: time.Date(2024, 6, 26, 3, 14, 45, 9, &time.Location{})},
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
				if tt.want.id.String() != actual.id.String() {
					t.Errorf("makeTraceMetaData() id: actual: \n%v\nwant: \n%v", actual, tt.want)
				}
				if tt.want.time.String() != actual.time.String() {
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
