package codec

import (
	"bytes"
	"encoding/binary"
	"math"
	"sync"

	"github.com/golang/protobuf/proto"
)

// Codec defines the codec specification for data
type Codec interface {
	Encode([]byte) ([]byte, error)
	Decode([]byte) ([]byte, error)
}

const FrameHeadLen = 15 //定义数据帧头大小
const Magic = 0x11      //定义魔数
const Version = 0       //当前版本

// 数据帧头
type FrameHeader struct {
	Magic        uint8  // 魔数  => 硬写到代码里的整数常量
	Version      uint8  // 版本号 用来支持版本迭代
	MsgType      uint8  // 消息类型 e.g. :   0x0: 普通消息 ,  0x1: 心跳消息
	ReqType      uint8  // 请求类型 e.g. :   0x0: 一发一收,   0x1: 只发不收,  0x2: 客户端流式请求, 0x3: 服务端流式请求, 0x4: 双向流式请求
	CompressType uint8  // 是否压缩 :  0x0: 不压缩,  0x1: 压缩
	StreamID     uint16 // 流 id 为了支持后续流式传输的能力
	Length       uint32 // 消息的长度
	Reserved     uint32 // 4个字节的保留位
}

// GetCodec get a Codec by a codec name
func GetCodec(name string) Codec {
	if codec, ok := codecMap[name]; ok {
		return codec
	}
	return DefaultCodec
}

var codecMap = make(map[string]Codec)

// DefaultCodec defines the default codec
var DefaultCodec = NewCodec()

// NewCodec returns a globally unique codec
var NewCodec = func() Codec {
	return &defaultCodec{}
}

func init() {
	RegisterCodec("proto", DefaultCodec)
}

// RegisterCodec registers a codec, which will be added to codecMap
func RegisterCodec(name string, codec Codec) {
	if codecMap == nil {
		codecMap = make(map[string]Codec)
	}
	codecMap[name] = codec
}

// 编码 => 将一个经过序列化的 request/response 二进制数据，拼接帧头形成一个完整的数据帧
func (c *defaultCodec) Encode(data []byte) ([]byte, error) {

	totalLen := FrameHeadLen + len(data)
	buffer := bytes.NewBuffer(make([]byte, 0, totalLen))

	//封装帧头
	frame := FrameHeader{
		Magic:        Magic,
		Version:      Version,
		MsgType:      0x0,
		ReqType:      0x0,
		CompressType: 0x0, // 默认不压缩
		Length:       uint32(len(data)),
	}

	// binary.BigEndian 大端序 => 网络传输一般是大端序

	// 拼装帧头
	if err := binary.Write(buffer, binary.BigEndian, frame.Magic); err != nil {
		return nil, err
	}
	if err := binary.Write(buffer, binary.BigEndian, frame.Version); err != nil {
		return nil, err
	}
	if err := binary.Write(buffer, binary.BigEndian, frame.MsgType); err != nil {
		return nil, err
	}
	if err := binary.Write(buffer, binary.BigEndian, frame.ReqType); err != nil {
		return nil, err
	}
	if err := binary.Write(buffer, binary.BigEndian, frame.CompressType); err != nil {
		return nil, err
	}
	if err := binary.Write(buffer, binary.BigEndian, frame.StreamID); err != nil {
		return nil, err
	}
	if err := binary.Write(buffer, binary.BigEndian, frame.Length); err != nil {
		return nil, err
	}
	if err := binary.Write(buffer, binary.BigEndian, frame.Reserved); err != nil {
		return nil, err
	}

	// 拼装包数据成为一个完整的数据帧
	if err := binary.Write(buffer, binary.BigEndian, data); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// 解码
func (c *defaultCodec) Decode(frame []byte) ([]byte, error) {
	//去掉帧头，就是包头+包体
	return frame[FrameHeadLen:], nil
}

type defaultCodec struct{}

func upperLimit(val int) uint32 {
	if val > math.MaxInt32 {
		return uint32(math.MaxInt32)
	}
	return uint32(val)
}

var bufferPool = &sync.Pool{
	New: func() interface{} {
		return &cachedBuffer{
			Buffer:            proto.Buffer{},
			lastMarshaledSize: 16,
		}
	},
}

type cachedBuffer struct {
	proto.Buffer
	lastMarshaledSize uint32
}
