package common

import (
	"context"
	"time"
)

// 连接接口
type IConn interface {
	Recv() ([]byte, error)
	RecvNonblock() ([]byte, error)
	Send([]byte) error
	SendNonblock(buf []byte) error
	SetTick(tick time.Duration)
	Run()
	Close()
	CloseWait(int)
}

// 服務器連接接口
type IServConn interface {
	IConn
	Wait(ctx context.Context) ([]byte, error)
}

// 会话接口
type ISession interface {
	Send([]byte) error
	Close()
	CloseWaitSecs(int)
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
