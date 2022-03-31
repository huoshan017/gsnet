package common

import (
	"context"
	"net"
	"time"

	"github.com/huoshan017/gsnet/common/packet"
)

// 连接接口
type IConn interface {
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	Recv() (packet.IPacket, error)
	RecvNonblock() (packet.IPacket, error)
	Send([]byte, bool) error
	SendPoolBuffer(*[]byte, packet.MemoryManagementType) error
	SendBytesArray([][]byte, bool) error
	SendPoolBufferArray([]*[]byte, packet.MemoryManagementType) error
	//SendNonblock([]byte, bool) error
	Run()
	Wait(ctx context.Context) (packet.IPacket, error)
	Close()
	CloseWait(int)
}

// 会话接口
type ISession interface {
	Send([]byte, bool) error
	SendBytesArray([][]byte, bool) error
	SendPoolBuffer(*[]byte, packet.MemoryManagementType) error
	SendPoolBufferArray([]*[]byte, packet.MemoryManagementType) error
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
	OnPacket(ISession, packet.IPacket) error
	OnTick(ISession, time.Duration)
	OnError(error)
}
