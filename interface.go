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
	SetTick(tick time.Duration)
	WaitSelect() ([]byte, error)
	Run()
	Close()
}

// 会话接口
type ISession interface {
	Send([]byte) error
	Close()
	GetId() uint64
	SetData(string, interface{})
	GetData(string) interface{}
}

// 会话处理器接口
type ISessionHandler interface {
	OnConnect(ISession)
	OnDisconnect(ISession, error)
	OnData(ISession, []byte) error
	OnTick(ISession, time.Duration)
	OnError(error)
}
