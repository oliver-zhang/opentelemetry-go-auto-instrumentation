package oneapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/alibaba/opentelemetry-go-auto-instrumentation/pkg/api"
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/relay/adaptor"
	"github.com/songquanpeng/one-api/relay/meta"
	"github.com/songquanpeng/one-api/relay/model"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func RelayOnEnter(call api.CallContext, c *gin.Context) {
	parentContext := c.Request.Context()
	ctx := BuildOneApiInstrumenter().Start(parentContext, oneApiRequest{})
	c.Set("ctx", ctx)
	data := make(map[string]interface{}, 1)
	data["ctx"] = ctx
	call.SetData(data)
	currentSpan := trace.SpanFromContext(ctx)
	currentSpan.SetAttributes(attribute.String("oneapi.tenant", c.GetHeader("APIKey")))
	currentSpan.SetAttributes(attribute.Int("oneapi.channel.type", c.GetInt(ctxkey.Channel)))
	currentSpan.SetAttributes(attribute.Int("oneapi.channel.id", c.GetInt(ctxkey.ChannelId)))
	currentSpan.SetAttributes(attribute.String("oneapi.channel.name", c.GetString(ctxkey.ChannelName)))
	currentSpan.SetAttributes(attribute.Int("oneapi.token.id", c.GetInt(ctxkey.TokenId)))
	currentSpan.SetAttributes(attribute.String("oneapi.token.name", c.GetString(ctxkey.TokenName)))
	currentSpan.SetAttributes(attribute.String("oneapi.group", c.GetString(ctxkey.Group)))
	currentSpan.SetAttributes(attribute.String("oneapi.origin.model.name", c.GetString(ctxkey.RequestModel)))
}

func RelayOnOnExit(call api.CallContext) {
	data, ok := call.GetData().(map[string]interface{})
	if !ok || data == nil || data["ctx"] == nil {
		return
	}
	ctx := data["ctx"].(context.Context)
	BuildOneApiInstrumenter().End(ctx, oneApiRequest{}, oneApiResponse{}, nil)
}

func GetRequestBodyOnEnter(call api.CallContext, c *gin.Context, meta *meta.Meta, textRequest *model.GeneralOpenAIRequest, adaptor adaptor.Adaptor) {
	ctxValue := c.Value("ctx")
	if ctx, ok := ctxValue.(context.Context); ok {
		currentSpan := trace.SpanFromContext(ctx)
		originalRequestBody, _ := json.Marshal(textRequest)
		currentSpan.SetAttributes(attribute.String("original.http.request.body", string(originalRequestBody)))
		targetRequest, _ := adaptor.ConvertRequest(c, meta.Mode, textRequest)
		targetRequestBody, _ := json.Marshal(targetRequest)
		currentSpan.SetAttributes(attribute.String("target.http.request.body", string(targetRequestBody)))
	}

}

func DoResponseOnEnter(call api.CallContext, adaptor interface{}, c *gin.Context, resp *http.Response, meta *meta.Meta) {
	ctxValue := c.Value("ctx")
	if ctx, ok := ctxValue.(context.Context); ok {
		data := make(map[string]interface{}, 1)
		data["ctx"] = ctx
		call.SetData(data)
		if meta.IsStream {
			return
		}
		responseBody, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		currentSpan := trace.SpanFromContext(ctx)
		currentSpan.SetAttributes(attribute.String("http.response.body", string(responseBody)))
		resp.Body = io.NopCloser(bytes.NewBuffer(responseBody))
	}
}

func DoResponseOnExit(call api.CallContext, usage *model.Usage, err *model.ErrorWithStatusCode) {
	data, ok := call.GetData().(map[string]interface{})
	if !ok || data == nil || data["ctx"] == nil {
		return
	}
	ctx := data["ctx"].(context.Context)
	currentSpan := trace.SpanFromContext(ctx)
	if err != nil {
		currentSpan.SetAttributes(attribute.String("exception.message", err.Message))
		currentSpan.SetAttributes(attribute.String("exception.type", err.Type))
		currentSpan.SetAttributes(attribute.Int("status.code", err.StatusCode))
	} else {
		currentSpan.SetAttributes(attribute.Int("oneapi.completion.tokens", usage.CompletionTokens))
		currentSpan.SetAttributes(attribute.Int("oneapi.prompt.tokens", usage.PromptTokens))
	}
}

func StreamHandlerOnEnter(call api.CallContext, c *gin.Context, resp *http.Response, relayMode int) {
	ctxValue := c.Value("ctx")
	if ctx, ok := ctxValue.(context.Context); ok {
		data := make(map[string]interface{}, 1)
		data["ctx"] = ctx
		call.SetData(data)
	}
}

func StreamHandlerOnExit(call api.CallContext, err *model.ErrorWithStatusCode, responseText string, usage *model.Usage) {
	data, ok := call.GetData().(map[string]interface{})
	if !ok || data == nil || data["ctx"] == nil {
		return
	}
	if err == nil {
		ctx := data["ctx"].(context.Context)
		currentSpan := trace.SpanFromContext(ctx)
		currentSpan.SetAttributes(attribute.String("http.response.body", responseText))
	}

}

func StringDataOnEnter(call api.CallContext, c *gin.Context, str string) {
	ctxValue := c.Value("ctx")
	if ctx, ok := ctxValue.(context.Context); ok {
		currentSpan := trace.SpanFromContext(ctx)
		currentSpan.AddEvent("response.stream", trace.WithAttributes(attribute.String("response.stream.text", str)))
	}
}
