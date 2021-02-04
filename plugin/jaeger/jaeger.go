package jaeger

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/lubanproj/gorpc/interceptor"
	"github.com/lubanproj/gorpc/plugin"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
	"github.com/uber/jaeger-client-go/config"
)

// Jaeger implements the opentracing specification
type Jaeger struct {
	opts *plugin.Options
}

const Name = "jaeger" //定义插件名称
const JaegerClientName = "gorpc-client-jaeger"
const JaegerServerName = "gorpc-server-jaeger"

func init() {
	plugin.Register(Name, JaegerSvr)
}

// global jaeger objects for framework
var JaegerSvr = &Jaeger{
	opts: &plugin.Options{},
}

// jaegerCarrier 是一种 map[string] []byte 结构，用来作为传输一些 key-value 数据的载体
type jaegerCarrier map[string][]byte

func (m jaegerCarrier) Set(key, val string) {
	key = strings.ToLower(key)
	m[key] = []byte(val)
}

func (m jaegerCarrier) ForeachKey(handler func(key, val string) error) error {
	for k, v := range m {
		handler(k, string(v))
	}
	return nil
}

// OpenTracingClientInterceptor client 端的拦截器
func OpenTracingClientInterceptor(tracer opentracing.Tracer, spanName string) interceptor.ClientInterceptor {

	return func(ctx context.Context, req, rsp interface{}, ivk interceptor.Invoker) error {

		//var parentCtx opentracing.SpanContext
		//
		////先通过 opentracing.SpanFromContext 获取上游带下来的 span 上下文信息
		//if parent := opentracing.SpanFromContext(ctx); parent != nil {
		//	parentCtx = parent.Context()
		//}

		//调用 tracer.StartSpan 创建一个 client span
		//clientSpan := tracer.StartSpan(spanName, ext.SpanKindRPCClient, opentracing.ChildOf(parentCtx))
		clientSpan := tracer.StartSpan(spanName, ext.SpanKindRPCClient)
		defer clientSpan.Finish()

		mdCarrier := &jaegerCarrier{}

		//通过调用 tracer.Inject，将所需要透传给下游的一些信息塞到 Span 里面
		if err := tracer.Inject(clientSpan.Context(), opentracing.HTTPHeaders, mdCarrier); err != nil {
			clientSpan.LogFields(log.String("event", "Tracer.Inject() failed"), log.Error(err))
		}

		clientSpan.LogFields(log.String("spanName", spanName))

		return ivk(ctx, req, rsp)

	}
}

// OpenTracingServerInterceptor 服务端的拦截器
func OpenTracingServerInterceptor(tracer opentracing.Tracer, spanName string) interceptor.ServerInterceptor {

	return func(ctx context.Context, req interface{}, handler interceptor.Handler) (interface{}, error) {

		mdCarrier := &jaegerCarrier{}

		//调用 tracer.Extract 解析 Span 的上下文信息，获得一个 SpanContext
		spanContext, err := tracer.Extract(opentracing.HTTPHeaders, mdCarrier)
		if err != nil && err != opentracing.ErrSpanContextNotFound {
			return nil, errors.New(fmt.Sprintf("tracer extract error : %v", err))
		}
		//调用 tracer.StartSpan 进行创建一个 server span
		serverSpan := tracer.StartSpan(spanName, ext.RPCServerOption(spanContext), ext.SpanKindRPCServer)
		defer serverSpan.Finish()

		//把 server span 放到上下文 context 中进行透传
		ctx = opentracing.ContextWithSpan(ctx, serverSpan)

		serverSpan.LogFields(log.String("spanName", spanName))

		return handler(ctx, req)
	}

}

// Init 在加载框架时实现 jaeger 配置的初始化 ( client 的初始化)
func Init(tracingSvrAddr string, opts ...plugin.Option) (opentracing.Tracer, error) {
	return initJaeger(tracingSvrAddr, JaegerClientName, opts...)
}

// server 的初始化
func (j *Jaeger) Init(opts ...plugin.Option) (opentracing.Tracer, error) {

	// config 设置
	for _, o := range opts {
		o(j.opts)
	}

	if j.opts.TracingSvrAddr == "" {
		return nil, errors.New("jaeger init error, traingSvrAddr is empty")
	}

	return initJaeger(j.opts.TracingSvrAddr, JaegerServerName, opts...)
}

// 初始化 Jaeger
func initJaeger(tracingSvrAddr string, jaegerServiceName string, opts ...plugin.Option) (opentracing.Tracer, error) {
	// 初始化 jaeger 的配置
	cfg := &config.Configuration{
		// 采样设置
		Sampler: &config.SamplerConfig{
			Type:  "const", // 固定采样
			Param: 1,       // 1= 全采样, 0= 不全采样
		},
		Reporter: &config.ReporterConfig{
			LogSpans:           true,
			LocalAgentHostPort: tracingSvrAddr,
		},
		ServiceName: jaegerServiceName,
	}

	// 通过配置来创建一个 tracer 实例
	tracer, _, err := cfg.NewTracer()
	if err != nil {
		return nil, err
	}

	//将 tracer 实例作为 opentracing 规范的实现
	opentracing.SetGlobalTracer(tracer)

	return tracer, err
}
