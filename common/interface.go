package common

import (
	"context"
	"net"
	"time"

	"github.com/huoshan017/gsnet/packet"
)

// 连接接口
type IConn interface {
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	Recv() (packet.IPacket, error)
	RecvNonblock() (packet.IPacket, error)
	Send(packet.PacketType, []byte, bool) error
	SendPoolBuffer(packet.PacketType, *[]byte, packet.MemoryManagementType) error
	SendBytesArray(packet.PacketType, [][]byte, bool) error
	SendPoolBufferArray(packet.PacketType, []*[]byte, packet.MemoryManagementType) error
	Wait(ctx context.Context) (packet.IPacket, error)
	Run()
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
	SetData(string, any)
	GetData(string) any
}

// 会话处理器接口
type ISessionEventHandler interface {
	OnConnect(ISession)
	OnPacket(ISession, packet.IPacket) error
	OnTick(ISession, time.Duration)
	OnDisconnect(ISession, error)
	OnError(error)
}

type ISessionEventHandlerEx interface {
	ISessionEventHandler
	OnChannel(any)
}
