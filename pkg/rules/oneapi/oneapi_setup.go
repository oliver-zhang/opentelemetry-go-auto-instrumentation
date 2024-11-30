package oneapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/alibaba/opentelemetry-go-auto-instrumentation/pkg/api"
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/relay/adaptor"
	"github.com/songquanpeng/one-api/relay/meta"
	"github.com/songquanpeng/one-api/relay/model"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"io"
	"net/http"
)

func RelayOnEnter(call api.CallContext, c *gin.Context) {
	parentContext := c.Request.Context()
	fmt.Printf("Pelay On Enter parentContext: %+v\n", parentContext)
	//currentSpan := trace.SpanFromContext(parentContext)
	//currentSpan.SetAttributes(attribute.String("relay.key", "relay.value"))
	//fmt.Printf("Pelay On Enter currentSpan: %+v\n", currentSpan)
	ctx := BuildOneApiInstrumenter().Start(parentContext, oneApiRequest{})
	c.Set("ctx", ctx)
	data := make(map[string]interface{}, 1)
	data["ctx"] = ctx
	call.SetData(data)
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
	fmt.Printf("Relay get request body on enter: %+v, %+v, %+v\n", meta, textRequest, adaptor)
	ctx := c.Value("ctx")
	fmt.Printf("Relay get request body ctx is :%v\n", ctx)
	currentSpan := trace.SpanFromContext(ctx.(context.Context))
	originalRequestBody, _ := json.Marshal(textRequest)
	currentSpan.SetAttributes(attribute.String("originalRequestBody", string(originalRequestBody)))
	targetRequest, _ := adaptor.ConvertRequest(c, meta.Mode, textRequest)
	targetRequestBody, _ := json.Marshal(targetRequest)
	currentSpan.SetAttributes(attribute.String("targetRequestBody", string(targetRequestBody)))
	data := make(map[string]interface{}, 1)
	data["ctx"] = ctx
	call.SetData(data)
}

func GetRequestBodyOnExit(call api.CallContext, reader io.Reader, err error) {

	data, ok := call.GetData().(map[string]interface{})
	if !ok || data == nil || data["ctx"] == nil {
		return
	}
	//ctx := data["ctx"].(context.Context)
	//requestBody, _ := io.ReadAll(reader)
	//err = reader.Close()
	//if err != nil {
	//}
	//reader = io.NopCloser(bytes.NewBuffer(requestBody))
	//currentSpan := trace.SpanFromContext(ctx)
	//currentSpan.SetAttributes(attribute.String("requestBody", string(requestBody)))
	fmt.Println("relay get request body on exit")
}

func DoResponseOnEnter(call api.CallContext, adaptor interface{}, c *gin.Context, resp *http.Response, meta *meta.Meta) {
	fmt.Printf("relay do response on enter: %+v, %+v, %+v\n", c, resp, meta)
	ctx := c.Value("ctx").(context.Context)
	responseBody, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	currentSpan := trace.SpanFromContext(ctx)
	currentSpan.SetAttributes(attribute.String("responseBody", string(responseBody)))
	resp.Body = io.NopCloser(bytes.NewBuffer(responseBody))
}

func DoResponseOnExit(call api.CallContext, usage *model.Usage, err *model.ErrorWithStatusCode) {
	fmt.Printf("relay do response  on exit %+v,%+v/n", usage, err)
	//c := call.GetParam(0).(gin.Context)
	//ctx := c.Value("ctx").(context.Context)
	//currentSpan := trace.SpanFromContext(ctx)
	//currentSpan.SetAttributes()
}
