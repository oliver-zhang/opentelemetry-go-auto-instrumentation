package oneapi

import (
	"context"
	"github.com/alibaba/opentelemetry-go-auto-instrumentation/pkg/inst-api/instrumenter"
	"go.opentelemetry.io/otel/attribute"
)

type oneApiSpanNameExtractor struct{}

func (o oneApiSpanNameExtractor) Extract(request oneApiRequest) string {
	return "relay.Relay"
}

type myTest[REQUEST any, RESPONSE any] struct {
}

func (m *myTest[REQUEST, RESPONSE]) OnStart(attributes []attribute.KeyValue, parentContext context.Context, request REQUEST) []attribute.KeyValue {
	attributes = append(attributes, attribute.KeyValue{
		Key:   attribute.Key("test-key"),
		Value: attribute.StringValue("TestValue"),
	})
	return attributes
}

func (m *myTest[REQUEST, RESPONSE]) OnEnd(attributes []attribute.KeyValue, context context.Context, request REQUEST, response RESPONSE, err error) []attribute.KeyValue {
	return attributes
}

func BuildOneApiInstrumenter() *instrumenter.InternalInstrumenter[oneApiRequest, oneApiResponse] {
	builder := &instrumenter.Builder[oneApiRequest, oneApiResponse]{}
	return builder.Init().SetSpanNameExtractor(&oneApiSpanNameExtractor{}).SetSpanKindExtractor(&instrumenter.AlwaysInternalExtractor[oneApiRequest]{}).
		AddAttributesExtractor(&myTest[oneApiRequest, oneApiResponse]{}).BuildInstrumenter()
}
