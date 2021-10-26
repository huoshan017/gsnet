package gsnet

import (
	"time"
	"unsafe"
)

// 选项结构
type Options struct {
	tickSpan      time.Duration
	tickHandle    func(time.Duration)
	DataProto     IDataProto
	MsgProto      IMsgProto
	SendChanLen   int
	RecvChanLen   int
	WriteBuffSize int
	ReadBuffSize  int
}

// 选项
type Option func(*Options)

func (options *Options) SetTickSpan(span time.Duration) {
	options.tickSpan = span
}

func (options *Options) SetTickHandle(handle func(time.Duration)) {
	options.tickHandle = handle
}

func (options *Options) SetDataProto(proto IDataProto) {
	options.DataProto = proto
}

func (options *Options) SetMsgProto(proto IMsgProto) {
	options.MsgProto = proto
}

func (options *Options) SetSendChanLen(chanLen int) {
	options.SendChanLen = chanLen
}

func (options *Options) SetRecvChanLen(chanLen int) {
	options.RecvChanLen = chanLen
}

func (options *Options) SetWriteBuffSize(size int) {
	options.WriteBuffSize = size
}

func (options *Options) SetReadBuffSize(size int) {
	options.ReadBuffSize = size
}

func SetTickSpan(span time.Duration) Option {
	return func(options *Options) {
		options.SetTickSpan(span)
	}
}

func SetDataProto(proto IDataProto) Option {
	return func(options *Options) {
		options.SetDataProto(proto)
	}
}

func SetMsgProto(proto IMsgProto) Option {
	return func(options *Options) {
		options.SetMsgProto(proto)
	}
}

func SetSendChanLen(chanLen int) Option {
	return func(options *Options) {
		options.SendChanLen = chanLen
	}
}

func SetRecvChanLen(chanLen int) Option {
	return func(options *Options) {
		options.RecvChanLen = chanLen
	}
}

func SetWriteBuffSize(size int) Option {
	return func(options *Options) {
		options.WriteBuffSize = size
	}
}

func SetReadBuffSize(size int) Option {
	return func(options *Options) {
		options.ReadBuffSize = size
	}
}

// 客户端选项结构
type ClientOptions struct {
	Options
}

// 服务选项结构
type ServiceOptions struct {
	Options
	ErrChanLen        int           // 错误通道长度
	SessionHandleTick time.Duration // 会话逻辑处理时间间隔
}

func (options *ServiceOptions) SetErrChanLen(length int) {
	options.ErrChanLen = length
}

func (options *ServiceOptions) SetSessionHandleTick(tick time.Duration) {
	options.SessionHandleTick = tick
}

func SetErrChanLen(length int) Option {
	return func(options *Options) {
		p := (*ServiceOptions)(unsafe.Pointer(options))
		p.SetErrChanLen(length)
	}
}

func SetSessionHandleTick(tick time.Duration) Option {
	return func(options *Options) {
		p := (*ServiceOptions)(unsafe.Pointer(options))
		p.SetSessionHandleTick(tick)
	}
}
