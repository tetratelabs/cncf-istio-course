package main

import (
	"github.com/valyala/fastjson"

	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
)

func main() {
	proxywasm.SetVMContext(&vmContext{})
}

// Override types.DefaultPluginContext.
func (ctx pluginContext) OnPluginStart(pluginConfigurationSize int) types.OnPluginStartStatus {
	data, err := proxywasm.GetPluginConfiguration()
	if err != nil {
		proxywasm.LogCriticalf("error reading plugin configuration: %v", err)
	}

	var p fastjson.Parser
	v, err := p.ParseBytes(data)
	if err != nil {
		proxywasm.LogCriticalf("error parsing configuration: %v", err)
	}

	obj, err := v.Object()
	if err != nil {
		proxywasm.LogCriticalf("error getting object from json value: %v", err)
	}

	obj.Visit(func(k []byte, v *fastjson.Value) {
		ctx.additionalHeaders[string(k)] = string(v.GetStringBytes())
	})

	return types.OnPluginStartStatusOK
}

type vmContext struct {
	// Embed the default VM context here,
	// so that we don't need to reimplement all the methods.
	types.DefaultVMContext
}

// Override types.DefaultVMContext.
func (*vmContext) NewPluginContext(contextID uint32) types.PluginContext {
	return &pluginContext{contextID: contextID, additionalHeaders: map[string]string{}}
}

type pluginContext struct {
	// Embed the default plugin context here,
	// so that we don't need to reimplement all the methods.
	types.DefaultPluginContext
	additionalHeaders map[string]string
	contextID         uint32
}

// Override types.DefaultPluginContext.
func (ctx *pluginContext) NewHttpContext(contextID uint32) types.HttpContext {
	proxywasm.LogInfo("NewHttpContext")
	return &httpContext{contextID: contextID, additionalHeaders: ctx.additionalHeaders}
}

type httpContext struct {
	// Embed the default http context here,
	// so that we don't need to reimplement all the methods.
	types.DefaultHttpContext
	contextID         uint32
	additionalHeaders map[string]string
}

func (ctx *httpContext) OnHttpResponseHeaders(numHeaders int, endOfStream bool) types.Action {
	proxywasm.LogInfo("OnHttpResponseHeaders")

	for key, value := range ctx.additionalHeaders {
		if err := proxywasm.AddHttpResponseHeader(key, value); err != nil {
			proxywasm.LogCriticalf("failed to add header: %v", err)
			return types.ActionPause
		}
		proxywasm.LogInfof("header set: %s=%s", key, value)
	}

	return types.ActionContinue
}
