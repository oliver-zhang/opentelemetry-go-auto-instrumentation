// Copyright (c) 2024 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package http

import (
	"context"
	"errors"
	"fmt"
	"github.com/alibaba/opentelemetry-go-auto-instrumentation/pkg/core/meter"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"log"
	"sync"
	"time"
)

const http_server_request_duration = "http.server.request.duration"

const http_client_request_duration = "http.client.request.duration"

type HttpServerMetric struct {
	key                   attribute.Key
	serverRequestDuration metric.Float64Histogram
}

type HttpClientMetric struct {
	key                   attribute.Key
	clientRequestDuration metric.Float64Histogram
}

var mu sync.Mutex
var httpServerOperationListener *HttpServerMetric
var httpClientOperationListener *HttpClientMetric
var shadower = httpMetricAttributesShadower{}

var httpMetricsConv = map[attribute.Key]bool{
	semconv.HTTPRequestMethodKey:      true,
	semconv.URLSchemeKey:              true,
	semconv.ErrorTypeKey:              true,
	semconv.HTTPResponseStatusCodeKey: true,
	semconv.HTTPRouteKey:              true,
	semconv.NetworkProtocolNameKey:    true,
	semconv.NetworkProtocolVersionKey: true,
	semconv.ServerAddressKey:          true,
	semconv.ServerPortKey:             true,
}

// InitHttpMetrics TODO: The init function may be executed after the HttpServerOperationListener() method
// so we need to make sure the otel_setup is executed before all the init() function
// related to issue https://github.com/alibaba/opentelemetry-go-auto-instrumentation/issues/48
func InitHttpMetrics(meter metric.Meter) {
	var err error
	httpServerOperationListener, err = newHttpServerMetric("net.http.server", meter)
	if err != nil {
		panic(err)
	}
	httpClientOperationListener, err = newHttpClientMetric("net.http.client", meter)
	if err != nil {
		panic(err)
	}
}

func HttpServerMetrics() *HttpServerMetric {
	mu.Lock()
	defer mu.Unlock()
	if httpServerOperationListener != nil {
		return httpServerOperationListener
	} else {
		return &HttpServerMetric{}
	}
}

func HttpClientMetrics() *HttpClientMetric {
	mu.Lock()
	defer mu.Unlock()
	if httpClientOperationListener != nil {
		return httpClientOperationListener
	} else {
		return &HttpClientMetric{}
	}
}

func newHttpServerMetric(key string, meter metric.Meter) (*HttpServerMetric, error) {
	m := &HttpServerMetric{
		key: attribute.Key(key),
	}
	d, err := newHttpServerRequestDurationMeasures(meter)
	if err != nil {
		return nil, err
	}
	m.serverRequestDuration = d
	return m, nil
}

func newHttpServerRequestDurationMeasures(meter metric.Meter) (metric.Float64Histogram, error) {
	mu.Lock()
	defer mu.Unlock()
	if meter == nil {
		return nil, errors.New("nil meter")
	}
	d, err := meter.Float64Histogram(http_server_request_duration,
		metric.WithUnit("ms"),
		metric.WithDescription("Duration of HTTP server requests."))
	if err == nil {
		return d, nil
	} else {
		return d, errors.New(fmt.Sprintf("failed to create http.server.request.duratio histogram, %v", err))
	}
}

func newHttpClientMetric(key string, meter metric.Meter) (*HttpClientMetric, error) {
	m := &HttpClientMetric{
		key: attribute.Key(key),
	}
	d, err := newHttpClientRequestDurationMeasures(meter)
	if err != nil {
		return nil, err
	}
	m.clientRequestDuration = d
	return m, nil
}

func newHttpClientRequestDurationMeasures(meter metric.Meter) (metric.Float64Histogram, error) {
	mu.Lock()
	defer mu.Unlock()
	if meter == nil {
		return nil, errors.New("nil meter")
	}
	d, err := meter.Float64Histogram(http_client_request_duration,
		metric.WithUnit("ms"),
		metric.WithDescription("Duration of HTTP client requests."))
	if err == nil {
		return d, nil
	} else {
		return d, errors.New(fmt.Sprintf("failed to create http.client.request.duratio histogram, %v", err))
	}
}

type httpMetricContext struct {
	startTime       time.Time
	startAttributes []attribute.KeyValue
}

func (h *HttpServerMetric) OnBeforeStart(parentContext context.Context, startTime time.Time) context.Context {
	return parentContext
}

func (h *HttpServerMetric) OnBeforeEnd(ctx context.Context, startAttributes []attribute.KeyValue, startTime time.Time) context.Context {
	return context.WithValue(ctx, h.key, httpMetricContext{
		startTime:       startTime,
		startAttributes: startAttributes,
	})
}

func (h *HttpServerMetric) OnAfterStart(context context.Context, endTime time.Time) {
	return
}

func (h *HttpServerMetric) OnAfterEnd(context context.Context, endAttributes []attribute.KeyValue, endTime time.Time) {
	mc := context.Value(h.key).(httpMetricContext)
	startTime, startAttributes := mc.startTime, mc.startAttributes
	// end attributes should be shadowed by AttrsShadower
	if h.serverRequestDuration == nil {
		var err error
		h.serverRequestDuration, err = newHttpServerRequestDurationMeasures(meter.GetMeter())
		if err != nil {
			log.Printf("failed to create serverRequestDuration, err is %v\n", err)
		}
	}
	endAttributes = append(endAttributes, startAttributes...)
	n, metricsAttrs := shadower.Shadow(endAttributes)
	if h.serverRequestDuration != nil {
		h.serverRequestDuration.Record(context, float64(endTime.Sub(startTime)), metric.WithAttributeSet(attribute.NewSet(metricsAttrs[0:n]...)))
	}
}

func (h HttpClientMetric) OnBeforeStart(parentContext context.Context, startTime time.Time) context.Context {
	return parentContext
}

func (h HttpClientMetric) OnBeforeEnd(ctx context.Context, startAttributes []attribute.KeyValue, startTime time.Time) context.Context {
	return context.WithValue(ctx, h.key, httpMetricContext{
		startTime:       startTime,
		startAttributes: startAttributes,
	})
}

func (h HttpClientMetric) OnAfterStart(context context.Context, endTime time.Time) {
	return
}

func (h HttpClientMetric) OnAfterEnd(context context.Context, endAttributes []attribute.KeyValue, endTime time.Time) {
	mc := context.Value(h.key).(httpMetricContext)
	startTime, startAttributes := mc.startTime, mc.startAttributes
	// end attributes should be shadowed by AttrsShadower
	if h.clientRequestDuration == nil {
		var err error
		// second change to init the metric
		h.clientRequestDuration, err = newHttpClientRequestDurationMeasures(meter.GetMeter())
		if err != nil {
			log.Printf("failed to create clientRequestDuration, err is %v\n", err)
		}
	}
	fmt.Printf("start attributes %v\n", startAttributes)
	endAttributes = append(endAttributes, startAttributes...)
	n, metricsAttrs := shadower.Shadow(endAttributes)
	if h.clientRequestDuration != nil {
		h.clientRequestDuration.Record(context, float64(endTime.Sub(startTime)), metric.WithAttributeSet(attribute.NewSet(metricsAttrs[0:n]...)))
	}
}

type httpMetricAttributesShadower struct{}

func (h httpMetricAttributesShadower) Shadow(attrs []attribute.KeyValue) (int, []attribute.KeyValue) {
	swap := func(attrs []attribute.KeyValue, i, j int) {
		tmp := attrs[i]
		attrs[i] = attrs[j]
		attrs[j] = tmp
	}
	index := 0
	for i, attr := range attrs {
		if _, ok := httpMetricsConv[attr.Key]; ok {
			if index != i {
				swap(attrs, i, index)
			}
			index++
		}
	}
	return index, attrs
}