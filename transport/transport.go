// 网络通信层，负责底层网络通信
// 主要包括 tcp 和 udp 两种协议的实现
package transport

import (
	"context"
	"encoding/binary"
	"io"
	"net"

	"github.com/lubanproj/gorpc/codec"
	"github.com/lubanproj/gorpc/codes"
)

const DefaultPayloadLength = 1024
const MaxPayloadLength = 4 * 1024 * 1024

// ServerTransport 定义所有 Server 传输层
// need to support
type ServerTransport interface {
	// 实现请求的监听和处理，所有的 server transport 都需要实现这个方法，
	// 同时设计成 interface 接口的方式，主要是为了实现可插拔，支持业务自定义
	ListenAndServe(context.Context, ...ServerTransportOption) error
}

// ClientTransport 定义所有 Client 传输层
// need to support
type ClientTransport interface {
	// 发起请求调用，传参除了上下文 context 之外，还有二进制的请求包 request，返回是一个二进制的完整数据帧
	Send(context.Context, []byte, ...ClientTransportOption) ([]byte, error)
}

// Framer 定义从数据流中读取数据帧
type Framer interface {
	// 读取数据帧的通用化定义
	ReadFrame(net.Conn) ([]byte, error)
}

type framer struct {
	buffer  []byte
	counter int // 防止死循环
}

// 创建一个数据帧
func NewFramer() Framer {
	return &framer{
		buffer: make([]byte, DefaultPayloadLength),
	}
}

//重新规划大小 会扩容成原来的两倍
func (f *framer) Resize() {
	f.buffer = make([]byte, len(f.buffer)*2)
}


func (f *framer) ReadFrame(conn net.Conn) ([]byte, error) {

	//读取出 15 byte 的帧头
	frameHeader := make([]byte, codec.FrameHeadLen)
	if num, err := io.ReadFull(conn, frameHeader); num != codec.FrameHeadLen || err != nil {
		return nil, err
	}

	// 验证魔数
	if magic := uint8(frameHeader[0]); magic != codec.Magic {
		return nil, codes.NewFrameworkError(codes.ClientMsgErrorCode, "invalid magic...")
	}

	//从帧头中获取包头 + 包体总长度 length ( 7~11 存储的是包的长度)
	length := binary.BigEndian.Uint32(frameHeader[7:11])

	if length > MaxPayloadLength {
		return nil, codes.NewFrameworkError(codes.ClientMsgErrorCode, "payload too large...")
	}

	//当 buffer > 4M 时或者 扩容的次数 counter 大于 12 时，会跳出循环，不再 Resize
	for uint32(len(f.buffer)) < length && f.counter <= 12 {
		f.buffer = make([]byte, len(f.buffer)*2)
		f.counter++
	}

	//读取 包头 + 包体
	if num, err := io.ReadFull(conn, f.buffer[:length]); uint32(num) != length || err != nil {
		return nil, err
	}

	return append(frameHeader, f.buffer[:length]...), nil
}
