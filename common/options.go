package common

import (
	"time"
)

// 选项结构
type Options struct {
	noDelay           bool
	keepAlived        bool
	keepAlivedPeriod  time.Duration
	readTimeout       time.Duration
	writeTimeout      time.Duration
	tickSpan          time.Duration
	dataProto         IDataProto
	sendChanLen       int
	recvChanLen       int
	writeBuffSize     int
	readBuffSize      int
	connCloseWaitSecs int // 連接關閉等待時間(秒)

	// todo 以下是需要实现的配置逻辑
	flushWriteInterval time.Duration // 写缓冲数据刷新到网络IO的最小时间间隔
	heartbeatInterval  time.Duration // 心跳间隔
}

// 选项
type Option func(*Options)

func (options *Options) GetNodelay() bool {
	return options.noDelay
}

func (options *Options) SetNodelay(noDelay bool) {
	options.noDelay = noDelay
}

func (options *Options) GetKeepAlived() bool {
	return options.keepAlived
}

func (options *Options) SetKeepAlived(keepAlived bool) {
	options.keepAlived = keepAlived
}

func (options *Options) GetKeepAlivedPeriod() time.Duration {
	return options.keepAlivedPeriod
}

func (options *Options) SetKeepAlivedPeriod(keepAlivedPeriod time.Duration) {
	options.keepAlivedPeriod = keepAlivedPeriod
}

func (options *Options) GetReadTimeout() time.Duration {
	return options.readTimeout
}

func (options *Options) SetReadTimeout(timeout time.Duration) {
	options.readTimeout = timeout
}

func (options *Options) GetWriteTimeout() time.Duration {
	return options.writeTimeout
}

func (options *Options) SetWriteTimeout(timeout time.Duration) {
	options.writeTimeout = timeout
}

func (options *Options) GetTickSpan() time.Duration {
	return options.tickSpan
}

func (options *Options) SetTickSpan(span time.Duration) {
	options.tickSpan = span
}

func (options *Options) GetDataProto() IDataProto {
	return options.dataProto
}

func (options *Options) SetDataProto(proto IDataProto) {
	options.dataProto = proto
}

func (options *Options) GetSendChanLen() int {
	return options.sendChanLen
}

func (options *Options) SetSendChanLen(chanLen int) {
	options.sendChanLen = chanLen
}

func (options *Options) GetRecvChanLen() int {
	return options.recvChanLen
}

func (options *Options) SetRecvChanLen(chanLen int) {
	options.recvChanLen = chanLen
}

func (options *Options) GetWriteBuffSize() int {
	return options.writeBuffSize
}

func (options *Options) SetWriteBuffSize(size int) {
	options.writeBuffSize = size
}

func (options *Options) GetReadBuffSize() int {
	return options.readBuffSize
}

func (options *Options) SetReadBuffSize(size int) {
	options.readBuffSize = size
}

func (options *Options) GetConnCloseWaitSecs() int {
	return options.connCloseWaitSecs
}

func (options *Options) SetConnCloseWaitSecs(secs int) {
	options.connCloseWaitSecs = secs
}

func WithNoDelay(noDelay bool) Option {
	return func(options *Options) {
		options.SetNodelay(noDelay)
	}
}

func WithKeepAlived(keepAlived bool) Option {
	return func(options *Options) {
		options.SetKeepAlived(keepAlived)
	}
}

func WithKeepAlivedPeriod(keepAlivedPeriod time.Duration) Option {
	return func(options *Options) {
		options.SetKeepAlivedPeriod(keepAlivedPeriod)
	}
}

func WithReadTimeout(timeout time.Duration) Option {
	return func(options *Options) {
		options.SetReadTimeout(timeout)
	}
}

func WithWriteTimeout(timeout time.Duration) Option {
	return func(options *Options) {
		options.SetWriteTimeout(timeout)
	}
}

func WithTickSpan(span time.Duration) Option {
	return func(options *Options) {
		options.SetTickSpan(span)
	}
}

func WithDataProto(proto IDataProto) Option {
	return func(options *Options) {
		options.SetDataProto(proto)
	}
}

func WithSendChanLen(chanLen int) Option {
	return func(options *Options) {
		options.SetSendChanLen(chanLen)
	}
}

func WithRecvChanLen(chanLen int) Option {
	return func(options *Options) {
		options.SetRecvChanLen(chanLen)
	}
}

func WithWriteBuffSize(size int) Option {
	return func(options *Options) {
		options.SetWriteBuffSize(size)
	}
}

func WithReadBuffSize(size int) Option {
	return func(options *Options) {
		options.SetReadBuffSize(size)
	}
}

func WithConnCloseWaitSecs(secs int) Option {
	return func(options *Options) {
		options.SetConnCloseWaitSecs(secs)
	}
}
