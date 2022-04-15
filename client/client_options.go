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

// 客户端选项结构
type ClientOptions struct {
	common.Options
	runMode RunMode
}

func (options *ClientOptions) GetRunMode() RunMode {
	return options.runMode
}

func (options *ClientOptions) SetRunMode(mode RunMode) {
	options.runMode = mode
}

func WithRunMode(mode RunMode) common.Option {
	return func(options *common.Options) {
		p := (*ClientOptions)(unsafe.Pointer(options))
		p.SetRunMode(mode)
	}
}
