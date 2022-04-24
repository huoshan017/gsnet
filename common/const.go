package common

import "time"

const (
	DefaultConnRecvChanLen = 100                   // 缺省连接接收通道长度
	DefaultConnSendChanLen = 100                   // 缺省连接发送通道长度
	DefaultReadTimeout     = time.Second * 5       // 缺省读超时
	DefaultWriteTimeout    = time.Second * 5       // 缺省写超时
	MinConnTick            = 10 * time.Millisecond // 连接中的最小tick
	DefaultConnTick        = 30 * time.Millisecond // 缺省tick
)
