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
	"github.com/alibaba/opentelemetry-go-auto-instrumentation/pkg/inst-api-semconv/instrumenter/net"
	"github.com/alibaba/opentelemetry-go-auto-instrumentation/pkg/inst-api/utils"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"strings"
)

// TODO: http.route

type HttpCommonAttrsExtractor[REQUEST any, RESPONSE any, GETTER1 HttpCommonAttrsGetter[REQUEST, RESPONSE], GETTER2 net.NetworkAttrsGetter[REQUEST, RESPONSE]] struct {
	HttpGetter       GETTER1
	NetGetter        GETTER2
	AttributesFilter func(attrs []attribute.KeyValue) []attribute.KeyValue
}

func (h *HttpCommonAttrsExtractor[REQUEST, RESPONSE, GETTER1, GETTER2]) OnStart(attributes []attribute.KeyValue, parentContext context.Context, request REQUEST) ([]attribute.KeyValue, context.Context) {
	attributes = append(attributes, attribute.KeyValue{
		Key:   semconv.HTTPRequestMethodKey,
		Value: attribute.StringValue(h.HttpGetter.GetRequestMethod(request)),
	})
	return attributes, parentContext
}

func (h *HttpCommonAttrsExtractor[REQUEST, RESPONSE, GETTER, GETTER2]) OnEnd(attributes []attribute.KeyValue, context context.Context, request REQUEST, response RESPONSE, err error) ([]attribute.KeyValue, context.Context) {
	statusCode := h.HttpGetter.GetHttpResponseStatusCode(request, response, err)
	protocolName := h.NetGetter.GetNetworkProtocolName(request, response)
	protocolVersion := h.NetGetter.GetNetworkProtocolVersion(request, response)
	attributes = append(attributes, attribute.KeyValue{
		Key:   semconv.HTTPResponseStatusCodeKey,
		Value: attribute.IntValue(statusCode),
	}, attribute.KeyValue{
		Key:   semconv.NetworkProtocolNameKey,
		Value: attribute.StringValue(protocolName),
	}, attribute.KeyValue{
		Key:   semconv.NetworkProtocolVersionKey,
		Value: attribute.StringValue(protocolVersion),
	})
	return attributes, context
}

type HttpClientAttrsExtractor[REQUEST any, RESPONSE any, GETTER1 HttpClientAttrsGetter[REQUEST, RESPONSE], GETTER2 net.NetworkAttrsGetter[REQUEST, RESPONSE]] struct {
	Base             HttpCommonAttrsExtractor[REQUEST, RESPONSE, GETTER1, GETTER2]
	NetworkExtractor net.NetworkAttrsExtractor[REQUEST, RESPONSE, GETTER2]
}

func (h *HttpClientAttrsExtractor[REQUEST, RESPONSE, GETTER1, GETTER2]) OnStart(attributes []attribute.KeyValue, parentContext context.Context, request REQUEST) ([]attribute.KeyValue, context.Context) {
	attributes, parentContext = h.Base.OnStart(attributes, parentContext, request)
	attributes, parentContext = h.NetworkExtractor.OnStart(attributes, parentContext, request)
	fullUrl := h.Base.HttpGetter.GetUrlFull(request)
	// TODO: add resend count
	attributes = append(attributes, attribute.KeyValue{
		Key:   semconv.URLFullKey,
		Value: attribute.StringValue(fullUrl),
	}, attribute.KeyValue{
		Key:   semconv.ServerAddressKey,
		Value: attribute.StringValue(h.Base.HttpGetter.GetServerAddress(request)),
	}, attribute.KeyValue{
		Key:   semconv.ServerPortKey,
		Value: attribute.IntValue(h.Base.HttpGetter.GetServerPort(request)),
	})
	if h.Base.AttributesFilter != nil {
		attributes = h.Base.AttributesFilter(attributes)
	}
	return attributes, parentContext
}

func (h *HttpClientAttrsExtractor[REQUEST, RESPONSE, GETTER1, GETTER2]) OnEnd(attributes []attribute.KeyValue, context context.Context, request REQUEST, response RESPONSE, err error) ([]attribute.KeyValue, context.Context) {
	attributes, context = h.Base.OnEnd(attributes, context, request, response, err)
	attributes, context = h.NetworkExtractor.OnEnd(attributes, context, request, response, err)
	if h.Base.AttributesFilter != nil {
		attributes = h.Base.AttributesFilter(attributes)
	}
	return attributes, context
}

func (h *HttpClientAttrsExtractor[REQUEST, RESPONSE, GETTER1, GETTER2]) GetSpanKey() attribute.Key {
	return utils.HTTP_CLIENT_KEY
}

type HttpServerAttrsExtractor[REQUEST any, RESPONSE any, GETTER1 HttpServerAttrsGetter[REQUEST, RESPONSE], GETTER2 net.NetworkAttrsGetter[REQUEST, RESPONSE], GETTER3 net.UrlAttrsGetter[REQUEST]] struct {
	Base             HttpCommonAttrsExtractor[REQUEST, RESPONSE, GETTER1, GETTER2]
	NetworkExtractor net.NetworkAttrsExtractor[REQUEST, RESPONSE, GETTER2]
	UrlExtractor     net.UrlAttrsExtractor[REQUEST, RESPONSE, GETTER3]
}

func (h *HttpServerAttrsExtractor[REQUEST, RESPONSE, GETTER1, GETTER2, GETTER3]) OnStart(attributes []attribute.KeyValue, parentContext context.Context, request REQUEST) ([]attribute.KeyValue, context.Context) {
	attributes, parentContext = h.Base.OnStart(attributes, parentContext, request)
	attributes, parentContext = h.UrlExtractor.OnStart(attributes, parentContext, request)
	userAgent := h.Base.HttpGetter.GetHttpRequestHeader(request, "User-Agent")
	var firstUserAgent string
	if len(userAgent) > 0 {
		firstUserAgent = userAgent[0]
	} else {
		firstUserAgent = ""
	}
	attributes = append(attributes, attribute.KeyValue{
		Key:   semconv.UserAgentOriginalKey,
		Value: attribute.StringValue(firstUserAgent),
	})
	if h.Base.AttributesFilter != nil {
		attributes = h.Base.AttributesFilter(attributes)
	}
	return attributes, parentContext
}

func (h *HttpServerAttrsExtractor[REQUEST, RESPONSE, GETTER1, GETTER2, GETTER3]) OnEnd(attributes []attribute.KeyValue, context context.Context, request REQUEST, response RESPONSE, err error) ([]attribute.KeyValue, context.Context) {
	attributes, context = h.Base.OnEnd(attributes, context, request, response, err)
	attributes, context = h.UrlExtractor.OnEnd(attributes, context, request, response, err)
	attributes, context = h.NetworkExtractor.OnEnd(attributes, context, request, response, err)
	span := trace.SpanFromContext(context)
	localRootSpan, ok := span.(sdktrace.ReadOnlySpan)
	if ok && span.IsRecording() {
		route := h.Base.HttpGetter.GetHttpRoute(request)
		if !strings.Contains(localRootSpan.Name(), route) {
			route = localRootSpan.Name()
		}
		attributes = append(attributes, attribute.KeyValue{
			Key:   semconv.HTTPRouteKey,
			Value: attribute.StringValue(route),
		})
	}
	if h.Base.AttributesFilter != nil {
		attributes = h.Base.AttributesFilter(attributes)
	}
	return attributes, context
}

func (h *HttpServerAttrsExtractor[REQUEST, RESPONSE, GETTER1, GETTER2, GETTER3]) GetSpanKey() attribute.Key {
	return utils.HTTP_SERVER_KEY
}
