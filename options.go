package gsnet

import (
	"time"
	"unsafe"
)

// 选项结构
type Options struct {
	tickSpan      time.Duration
	tickHandle    func(time.Duration)
	dataProto     IDataProto
	msgProto      IMsgProto
	sendChanLen   int
	recvChanLen   int
	writeBuffSize int
	readBuffSize  int
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
	options.dataProto = proto
}

func (options *Options) SetMsgProto(proto IMsgProto) {
	options.msgProto = proto
}

func (options *Options) SetSendChanLen(chanLen int) {
	options.sendChanLen = chanLen
}

func (options *Options) SetRecvChanLen(chanLen int) {
	options.recvChanLen = chanLen
}

func (options *Options) SetWriteBuffSize(size int) {
	options.writeBuffSize = size
}

func (options *Options) SetReadBuffSize(size int) {
	options.readBuffSize = size
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
		options.sendChanLen = chanLen
	}
}

func SetRecvChanLen(chanLen int) Option {
	return func(options *Options) {
		options.recvChanLen = chanLen
	}
}

func SetWriteBuffSize(size int) Option {
	return func(options *Options) {
		options.writeBuffSize = size
	}
}

func SetReadBuffSize(size int) Option {
	return func(options *Options) {
		options.readBuffSize = size
	}
}

// 客户端选项结构
type ClientOptions struct {
	Options
}

// 会话处理器函数类型
type NewSessionHandlerFunc func(args ...interface{}) ISessionHandler

// 服务选项结构
type ServiceOptions struct {
	Options
	createHandlerFunc     NewSessionHandlerFunc // 会话处理器创建函数
	createHandlerFuncArgs []interface{}         // 会话处理器创建函数参数列表
	errChanLen            int                   // 错误通道长度
	sessionHandleTick     time.Duration         // 会话逻辑处理时间间隔
}

func (options *ServiceOptions) SetNewSessionHandlerFunc(fun NewSessionHandlerFunc) {
	options.createHandlerFunc = fun
}

func (options *ServiceOptions) SetNewSessionHandlerFuncArgs(args ...interface{}) {
	options.createHandlerFuncArgs = args
}

func (options *ServiceOptions) SetErrChanLen(length int) {
	options.errChanLen = length
}

func (options *ServiceOptions) SetSessionHandleTick(tick time.Duration) {
	options.sessionHandleTick = tick
}

func SetNewSessionHandlerFunc(fun NewSessionHandlerFunc) Option {
	return func(options *Options) {
		p := (*ServiceOptions)(unsafe.Pointer(options))
		p.SetNewSessionHandlerFunc(fun)
	}
}

func SetNewSessionHandlerFuncArgs(args ...interface{}) Option {
	return func(options *Options) {
		p := (*ServiceOptions)(unsafe.Pointer(options))
		p.SetNewSessionHandlerFuncArgs(args...)
	}
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
