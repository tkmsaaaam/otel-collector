package traceBufferRedis

import (
	"context"
	"log"

	"github.com/tkmsaaaam/otel-collector/processor/traceBufferRedis/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
)

func NewFactory() processor.Factory {
	return processor.NewFactory(
		metadata.Type,
		createDefaultConfig,
		processor.WithTraces(createProcessor, metadata.TracesStability))
}

func createProcessor(context context.Context, settings processor.Settings, config component.Config, nextConsumer consumer.Traces) (processor.Traces, error) {
	c := config.(*Config)
	err := c.Validate()
	if err != nil {
		log.Println("settings invalid", err)
		return nil, err
	}
	return newTraceBuffer(context, c, nextConsumer)
}
