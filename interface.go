package gsnet

import (
	"time"
)

// 连接接口
type IConn interface {
	Recv() ([]byte, error)
	RecvNonblock() ([]byte, error)
	Send([]byte) error
	SendNonblock(buf []byte) error
	SetRecvDeadline(deadline time.Time)
	SetSendDeadline(deadline time.Time)
	Run()
	Close()
}

// 服务回调接口
type IServiceCallback interface {
	OnConnect(sessionId uint64)
	OnDisconnect(sessionId uint64, err error)
	OnError(err error)
	OnTick(tick time.Duration)
}

// 客户端回调接口
type IClientCallback interface {
	OnConnect()
	OnDisconnect(err error)
	OnError(err error)
	OnTick(tick time.Duration)
}

type IServiceHandler interface {
	OnData(*Session, []byte) error
}

// 处理器接口
type IClientHandler interface {
	OnData([]byte) error
}
