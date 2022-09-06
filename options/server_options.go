package options

import (
	"time"
	"unsafe"
)

// 会话处理器函数类型
//type NewSessionHandlerFunc func(args ...any) common.ISessionEventHandler

// 服务选项结构
type ServerOptions struct {
	Options
	connMaxCount          int           // 連接最大數
	connChanLen           int           // 連接通道長度
	createHandlerFuncArgs []any         // 会话处理器创建函数参数列表
	reusePort             bool          // 重用端口
	reuseAddr             bool          // 重用地址
	errChanLen            int           // 错误通道长度
	sessionHandleTick     time.Duration // 会话逻辑处理时间间隔
}

func (options *ServerOptions) GetConnMaxCount() int {
	return options.connMaxCount
}

func (options *ServerOptions) SetConnMaxCount(count int) {
	options.connMaxCount = count
}

func (options *ServerOptions) GetConnChanLen() int {
	return options.connChanLen
}

func (options *ServerOptions) SetConnChanLen(chanLen int) {
	options.connChanLen = chanLen
}

func (options *ServerOptions) GetNewSessionHandlerFuncArgs() []any {
	return options.createHandlerFuncArgs
}

func (options *ServerOptions) SetNewSessionHandlerFuncArgs(args ...any) {
	options.createHandlerFuncArgs = args
}

func (options *ServerOptions) GetReuseAddr() bool {
	return options.reuseAddr
}

func (options *ServerOptions) SetReuseAddr(enable bool) {
	options.reuseAddr = enable
}

func (options *ServerOptions) GetReusePort() bool {
	return options.reusePort
}

func (options *ServerOptions) SetReusePort(enable bool) {
	options.reusePort = enable
}

func (options *ServerOptions) GetErrChanLen() int {
	return options.errChanLen
}

func (options *ServerOptions) SetErrChanLen(length int) {
	options.errChanLen = length
}

func (options *ServerOptions) GetSessionHandleTick() time.Duration {
	return options.sessionHandleTick
}

func (options *ServerOptions) SetSessionHandleTick(tick time.Duration) {
	options.sessionHandleTick = tick
}

func WithConnMaxCount(count int) Option {
	return func(options *Options) {
		p := (*ServerOptions)(unsafe.Pointer(options))
		p.SetConnMaxCount(count)
	}
}

func WithConnChanLen(chanLen int) Option {
	return func(options *Options) {
		p := (*ServerOptions)(unsafe.Pointer(options))
		p.SetConnChanLen(chanLen)
	}
}

func WithNewSessionHandlerFuncArgs(args ...any) Option {
	return func(options *Options) {
		p := (*ServerOptions)(unsafe.Pointer(options))
		p.SetNewSessionHandlerFuncArgs(args...)
	}
}

func WithReuseAddr(enable bool) Option {
	return func(options *Options) {
		p := (*ServerOptions)(unsafe.Pointer(options))
		p.SetReuseAddr(enable)
	}
}

func WithReusePort(enable bool) Option {
	return func(options *Options) {
		p := (*ServerOptions)(unsafe.Pointer(options))
		p.SetReusePort(enable)
	}
}

func WithErrChanLen(length int) Option {
	return func(options *Options) {
		p := (*ServerOptions)(unsafe.Pointer(options))
		p.SetErrChanLen(length)
	}
}

func WithSessionHandleTick(tick time.Duration) Option {
	return func(options *Options) {
		p := (*ServerOptions)(unsafe.Pointer(options))
		p.SetSessionHandleTick(tick)
	}
}
