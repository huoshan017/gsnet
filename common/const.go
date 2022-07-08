package common

import "time"

const (
	DefaultConnRecvListLen = 100                   // 缺省连接接收队列长度
	DefaultConnSendListLen = 100                   // 缺省连接发送队列长度
	DefaultReadTimeout     = time.Second * 5       // 缺省读超时
	DefaultWriteTimeout    = time.Second * 5       // 缺省写超时
	MinConnTick            = 10 * time.Millisecond // 连接中的最小tick
	DefaultConnTick        = 30 * time.Millisecond // 缺省tick
)
