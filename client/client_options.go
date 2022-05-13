package client

import (
	"unsafe"

	"github.com/huoshan017/gsnet/common"
)

type RunMode int32

const (
	RunModeAsMainLoop RunMode = iota
	RunModeOnlyUpdate RunMode = 1
)

const (
	DefaultReconnectSeconds int32 = 5
)

// 客户端选项结构
type ClientOptions struct {
	common.Options
	runMode          RunMode
	reconnect        bool
	reconnectSeconds int32 // second
}

func (options *ClientOptions) GetRunMode() RunMode {
	return options.runMode
}

func (options *ClientOptions) SetRunMode(mode RunMode) {
	options.runMode = mode
}

func (options *ClientOptions) IsReconnect() bool {
	return options.reconnect
}

func (options *ClientOptions) SetReconnect(enable bool) {
	options.reconnect = enable
}

func (options *ClientOptions) GetReconnectSeconds() int32 {
	secs := options.reconnectSeconds
	if secs <= 0 {
		secs = DefaultReconnectSeconds
	}
	return secs
}

func (options *ClientOptions) SetReconnectSeconds(secs int32) {
	options.reconnectSeconds = secs
}

func WithRunMode(mode RunMode) common.Option {
	return func(options *common.Options) {
		p := (*ClientOptions)(unsafe.Pointer(options))
		p.SetRunMode(mode)
	}
}

func WithReconnect(enable bool) common.Option {
	return func(options *common.Options) {
		p := (*ClientOptions)(unsafe.Pointer(options))
		p.SetReconnect(enable)
	}
}

func WithReconnectSeconds(secs int32) common.Option {
	return func(options *common.Options) {
		p := (*ClientOptions)(unsafe.Pointer(options))
		p.SetReconnectSeconds(secs)
	}
}
