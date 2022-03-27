package common

import (
	"context"
	"time"
)

// 连接接口
type IConn interface {
	Recv() (interface{}, error)
	RecvNonblock() (interface{}, error)
	Send(interface{}) error
	SendNonblock(interface{}) error
	Run()
	Wait(ctx context.Context) (interface{}, error)
	Close()
	CloseWait(int)
}

// 会话接口
type ISession interface {
	Send(interface{}) error
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
	OnData(ISession, interface{}) error
	OnTick(ISession, time.Duration)
	OnError(error)
}
