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
//go:build ignore

package fasthttp

import (
	"github.com/valyala/fasthttp"
	"net/url"

	"time"
)

var fastHttpServerInstrumenter = BuildFastHttpServerOtelInstrumenter()

func newFastHttpServerDelegateHandler(handler fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		startTime := time.Now()
		handler(ctx)
		u, err := url.Parse(ctx.URI().String())
		if err != nil {
			return
		}
		request := fastHttpRequest{
			method: string(ctx.Method()),
			url:    u,
			isTls:  ctx.IsTLS(),
			header: &ctx.Request.Header,
		}
		fastHttpServerInstrumenter.StartAndEnd(ctx, request, fastHttpResponse{
			statusCode: ctx.Response.StatusCode(),
			header:     &ctx.Response.Header,
		}, ctx.Err(), startTime, time.Now())
	}
}

func listenAndServeFastHttpOnEnter(call fasthttp.CallContext, s *fasthttp.Server, addr string) {
	if s == nil {
		return
	}
	s.Handler = newFastHttpServerDelegateHandler(s.Handler)
	call.SetParam(0, s)
}