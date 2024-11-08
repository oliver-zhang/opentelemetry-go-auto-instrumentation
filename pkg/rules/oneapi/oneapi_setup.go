package oneapi

import (
	"context"
	"github.com/alibaba/opentelemetry-go-auto-instrumentation/pkg/api"
	"reflect"
	"strconv"
)

func getAllChannelsOnEnter(call api.CallContext,startIdx int, num int, scope string)  {
	println("startIdx: "+ strconv.Itoa(startIdx))
	println("num: "+ strconv.Itoa(num))
	println("scope: "+ scope)
	ctx := BuildOneApiInstrumenter().Start(context.Background(),oneApiRequest{})
	data := make(map[string]interface{}, 1)
	data["ctx"] = ctx
	call.SetData(data)
}


func getAllChannelsOnExit(call api.CallContext,err error)  {
	data, ok := call.GetData().(map[string]interface{})
	if !ok || data == nil || data["ctx"] == nil {
		return
	}
	ctx := data["ctx"].(context.Context)
	rt := call.GetReturnVal(0)
	println("result: "+reflect.TypeOf(rt).String())
	BuildOneApiInstrumenter().End(ctx,oneApiRequest{},oneApiResponse{},err)
}