// Copyright 2020-2021 Tetrate
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"strings"

	"github.com/tidwall/gjson"
	"golang.org/x/exp/slices"

	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
)

func main() {
	proxywasm.SetVMContext(&vmContext{})
}

type vmContext struct {
	// Embed the default VM context here,
	// so that we don't need to reimplement all the methods.
	types.DefaultVMContext
}

// Override types.DefaultVMContext.
func (*vmContext) NewPluginContext(contextID uint32) types.PluginContext {
	return &pluginContext{
		default_allowed_in_headers: []string{":authority",
			":method",
			":path",
			":scheme",
			"accept",
			"accept-encoding",
			"accept-language",
			"cache-control",
			"content-length",
			"content-type",
			"cookie",
			"dnt",
			"origin",
			"pragma",
			"referer",
			"sec-fetch-dest",
			"sec-fetch-mode",
			"sec-fetch-site",
			"sec-fetch-user",
			"upgrade-insecure-requests",
			"user-agent",
			"x-forwarded-for",
			"x-forwarded-proto",
			"x-request-id",
			"x-envoy-decorator-operation",
			"x-envoy-peer-metadata",
			"x-envoy-peer-metadata",
			"x-envoy-peer-metadata-id",
		},
		default_allowed_out_headers: []string{
			":status",
			"access-control-allow-credentials",
			"access-control-allow-headers",
			"access-control-allow-methods",
			"access-control-allow-origin",
			"access-control-allow-private-network",
			"access-control-expose-headers",
			"access-control-max-age",
			"age",
			"cache-control",
			"connection",
			"content-encoding",
			"content-length",
			"content-type",
			"date",
			"etag",
			"expires",
			"grpc-message",
			"grpc-status",
			"keep-alive",
			"last-modified",
			"location",
			"proxy-connection",
			"proxy-status",
			"server",
			"transfer-encoding",
			"upgrade",
			"vary",
			"via",
			"x-envoy-attempt-count",
			"x-envoy-decorator-operation",
			"x-envoy-degraded",
			"x-envoy-immediate-health-check-fail",
			"x-envoy-ratelimited",
			"x-envoy-upstream-canary",
			"x-envoy-upstream-healthchecked-cluster",
			"x-envoy-upstream-service-time",
			"x-request-id",
		},
	}
}

type pluginContext struct {
	// Embed the default plugin context here,
	// so that we don't need to reimplement all the methods.
	types.DefaultPluginContext
	default_allowed_in_headers  []string
	cfg_allowed_in_headers      []string
	default_allowed_out_headers []string
	cfg_allowed_out_headers     []string
}

// Override types.DefaultPluginContext.
func (p *pluginContext) NewHttpContext(contextID uint32) types.HttpContext {
	return &httpHeaders{
		contextID:                   contextID,
		default_allowed_in_headers:  p.default_allowed_in_headers,
		cfg_allowed_in_headers:      p.cfg_allowed_in_headers,
		default_allowed_out_headers: p.default_allowed_out_headers,
		cfg_allowed_out_headers:     p.cfg_allowed_out_headers,
	}
}

func (p *pluginContext) OnPluginStart(pluginConfigurationSize int) types.OnPluginStartStatus {
	proxywasm.LogDebug("loading plugin config")
	data, err := proxywasm.GetPluginConfiguration()
	if data == nil {
		return types.OnPluginStartStatusOK
	}

	if err != nil {
		proxywasm.LogCriticalf("error reading plugin configuration: %v", err)
		return types.OnPluginStartStatusFailed
	}

	if !gjson.Valid(string(data)) {
		proxywasm.LogCritical(`invalid configuration format, needs json format`)
		return types.OnPluginStartStatusFailed
	}

	gjson.Get(string(data), "in").ForEach(func(_, value gjson.Result) bool {
		p.cfg_allowed_in_headers = append(p.cfg_allowed_in_headers, strings.TrimSpace(value.Str))
		return true
	})

	gjson.Get(string(data), "out").ForEach(func(_, value gjson.Result) bool {
		p.cfg_allowed_out_headers = append(p.cfg_allowed_out_headers, strings.TrimSpace(value.Str))
		return true
	})

	proxywasm.LogInfof("cfg_allowed_in_headers from config: %v", p.cfg_allowed_in_headers)
	proxywasm.LogInfof("cfg_allowed_out_headers from config: %v", p.cfg_allowed_out_headers)

	return types.OnPluginStartStatusOK
}

type httpHeaders struct {
	// Embed the default http context here,
	// so that we don't need to reimplement all the methods.
	types.DefaultHttpContext
	contextID                   uint32
	default_allowed_in_headers  []string
	cfg_allowed_in_headers      []string
	default_allowed_out_headers []string
	cfg_allowed_out_headers     []string
}

// Override types.DefaultHttpContext.
func (ctx *httpHeaders) OnHttpRequestHeaders(numHeaders int, endOfStream bool) types.Action {
	hs, err := proxywasm.GetHttpRequestHeaders()
	if err != nil {
		proxywasm.LogCriticalf("failed to get request headers: %v", err)
	}

	for _, h := range hs {
		key := h[0]
		if !slices.Contains(ctx.default_allowed_in_headers, key) && !slices.Contains(ctx.cfg_allowed_in_headers, key) {
			err := proxywasm.RemoveHttpRequestHeader(key)
			if err != nil {
				proxywasm.LogCriticalf("Remove http request header failed key: %v", key)
				return types.ActionPause
			}
		}
	}
	return types.ActionContinue
}

// Override types.DefaultHttpContext.
func (ctx *httpHeaders) OnHttpResponseHeaders(_ int, _ bool) types.Action {
	hs, err := proxywasm.GetHttpResponseHeaders()
	if err != nil {
		proxywasm.LogCriticalf("failed to get response headers: %v", err)
	}

	for _, h := range hs {
		key := h[0]
		if !slices.Contains(ctx.default_allowed_out_headers, key) && !slices.Contains(ctx.cfg_allowed_out_headers, key) {
			err := proxywasm.RemoveHttpResponseHeader(key)
			if err != nil {
				proxywasm.LogCriticalf("Remove http response header failed key: %v", key)
				return types.ActionPause
			}
		}
	}
	return types.ActionContinue
}

// Override types.DefaultHttpContext.
func (ctx *httpHeaders) OnHttpStreamDone() {
	proxywasm.LogInfof("%d finished", ctx.contextID)
}
