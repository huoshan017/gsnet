package gsnet

import (
	"time"
	"unsafe"
)

// 选项结构
type Options struct {
	readTimeout   time.Duration
	writeTimeout  time.Duration
	tickSpan      time.Duration
	tickHandle    func(time.Duration)
	dataProto     IDataProto
	msgProto      IMsgProto
	sendChanLen   int
	recvChanLen   int
	writeBuffSize int
	readBuffSize  int

	// todo 以下是需要实现的配置逻辑
	flushWriteInterval       time.Duration // 写缓冲数据刷新到网络IO的最小时间间隔
	gracefulCloseWaitingTime time.Duration // 优雅关闭等待时间
	heartbeatInterval        time.Duration // 心跳间隔
}

// 选项
type Option func(*Options)

func (options *Options) SetReadTimeout(timeout time.Duration) {
	options.readTimeout = timeout
}

func (options *Options) SetWriteTimeout(timeout time.Duration) {
	options.writeTimeout = timeout
}

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

func SetReadTimeout(timeout time.Duration) Option {
	return func(options *Options) {
		options.SetReadTimeout(timeout)
	}
}

func SetWriteTimeout(timeout time.Duration) Option {
	return func(options *Options) {
		options.SetWriteTimeout(timeout)
	}
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
		options.SetSendChanLen(chanLen)
	}
}

func SetRecvChanLen(chanLen int) Option {
	return func(options *Options) {
		options.SetRecvChanLen(chanLen)
	}
}

func SetWriteBuffSize(size int) Option {
	return func(options *Options) {
		options.SetWriteBuffSize(size)
	}
}

func SetReadBuffSize(size int) Option {
	return func(options *Options) {
		options.SetReadBuffSize(size)
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
	connChanLen           int           // 連接通道長度
	createHandlerFuncArgs []interface{} // 会话处理器创建函数参数列表
	reusePort             bool          // 重用端口
	reuseAddr             bool          // 重用地址
	errChanLen            int           // 错误通道长度
	sessionHandleTick     time.Duration // 会话逻辑处理时间间隔
}

func (options *ServiceOptions) SetConnChanLen(chanLen int) {
	options.connChanLen = chanLen
}

func (options *ServiceOptions) SetNewSessionHandlerFuncArgs(args ...interface{}) {
	options.createHandlerFuncArgs = args
}

func (options *ServiceOptions) SetReuseAddr(enable bool) {
	options.reuseAddr = enable
}

func (options *ServiceOptions) SetReusePort(enable bool) {
	options.reusePort = enable
}

func (options *ServiceOptions) SetErrChanLen(length int) {
	options.errChanLen = length
}

func (options *ServiceOptions) SetSessionHandleTick(tick time.Duration) {
	options.sessionHandleTick = tick
}

func SetConnChanLen(chanLen int) Option {
	return func(options *Options) {
		p := (*ServiceOptions)(unsafe.Pointer(options))
		p.SetConnChanLen(chanLen)
	}
}

func SetNewSessionHandlerFuncArgs(args ...interface{}) Option {
	return func(options *Options) {
		p := (*ServiceOptions)(unsafe.Pointer(options))
		p.SetNewSessionHandlerFuncArgs(args...)
	}
}

func SetReuseAddr(enable bool) Option {
	return func(options *Options) {
		p := (*ServiceOptions)(unsafe.Pointer(options))
		p.SetReuseAddr(enable)
	}
}

func SetReusePort(enable bool) Option {
	return func(options *Options) {
		p := (*ServiceOptions)(unsafe.Pointer(options))
		p.SetReusePort(enable)
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
