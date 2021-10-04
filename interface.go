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

// 处理器接口
type IHandler interface {
	OnConnect(ISession)
	OnDisconnect(ISession, error)
	OnData(ISession, []byte) error
	OnTick(ISession, time.Duration)
	OnError(error)
}
