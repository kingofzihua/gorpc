package gorpc

import (
	"time"

	"github.com/lubanproj/gorpc/interceptor"
)

// ServerOptions defines the server serve parameters
type ServerOptions struct {
	address           string        // 监听地址 例子 :( ip://127.0.0.1:8080、 dns://www.google.com)
	network           string        // 网络类型 例子 : tcp、udp
	protocol          string        // 协议 例子 : proto、json
	timeout           time.Duration // 超时时间 timeout
	serializationType string        // 序列化类型 , default: proto

	selectorSvrAddr string   // service discovery server address, required when using the third-party service discovery plugin
	tracingSvrAddr  string   // tracing plugin server address, required when using the third-party tracing plugin
	tracingSpanName string   // tracing span name, required when using the third-party tracing plugin
	pluginNames     []string // plugin name
	interceptors    []interceptor.ServerInterceptor
}

type ServerOption func(*ServerOptions)

func WithAddress(address string) ServerOption {
	return func(o *ServerOptions) {
		o.address = address
	}
}

func WithNetwork(network string) ServerOption {
	return func(o *ServerOptions) {
		o.network = network
	}
}

func WithProtocol(protocol string) ServerOption {
	return func(o *ServerOptions) {
		o.protocol = protocol
	}
}

func WithTimeout(timeout time.Duration) ServerOption {
	return func(o *ServerOptions) {
		o.timeout = timeout
	}
}

func WithSerializationType(serializationType string) ServerOption {
	return func(o *ServerOptions) {
		o.serializationType = serializationType
	}
}

func WithSelectorSvrAddr(addr string) ServerOption {
	return func(o *ServerOptions) {
		o.selectorSvrAddr = addr
	}
}

func WithPlugin(pluginName ...string) ServerOption {
	return func(o *ServerOptions) {
		o.pluginNames = append(o.pluginNames, pluginName...)
	}
}

func WithInterceptor(interceptors ...interceptor.ServerInterceptor) ServerOption {
	return func(o *ServerOptions) {
		o.interceptors = append(o.interceptors, interceptors...)
	}
}

func WithTracingSvrAddr(addr string) ServerOption {
	return func(o *ServerOptions) {
		o.tracingSvrAddr = addr
	}
}

func WithTracingSpanName(name string) ServerOption {
	return func(o *ServerOptions) {
		o.tracingSpanName = name
	}
}
