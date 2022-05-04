package common

import (
	"context"
	"net"
	"time"

	"github.com/huoshan017/gsnet/packet"
)

type IdWithPacket struct {
	id  int32
	pak packet.IPacket
}

func (ip *IdWithPacket) Set(id int32, pak packet.IPacket) {
	ip.id = id
	ip.pak = pak
}

func (ip *IdWithPacket) GetId() int32 {
	return ip.id
}

func (ip *IdWithPacket) GetPak() packet.IPacket {
	return ip.pak
}

// 连接接口
type IConn interface {
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	Run()
	Close()
	CloseWait(int)
	Recv() (packet.IPacket, error)
	RecvNonblock() (packet.IPacket, error)
	Send(packet.PacketType, []byte, bool) error
	SendPoolBuffer(packet.PacketType, *[]byte, packet.MemoryManagementType) error
	SendBytesArray(packet.PacketType, [][]byte, bool) error
	SendPoolBufferArray(packet.PacketType, []*[]byte, packet.MemoryManagementType) error
	Wait(ctx context.Context, chPak chan IdWithPacket) (packet.IPacket, int32, error)
}

// 会话接口
type ISession interface {
	GetId() uint64
	Send([]byte, bool) error
	SendBytesArray([][]byte, bool) error
	SendPoolBuffer(*[]byte) error
	SendPoolBufferArray([]*[]byte) error
	Close()
	CloseWaitSecs(int)
	AddInboundHandle(int32, func(ISession, packet.IPacket) error)
	RemoveInboundHandle(int32)
	GetInboundHandles() map[int32]func(ISession, packet.IPacket) error
	GetPacketChannel() chan IdWithPacket
	SetUserData(string, any)
	GetUserData(string) any
}

// 会话处理器接口
type ISessionEventHandler interface {
	OnConnect(ISession)
	OnReady(ISession)
	OnPacket(ISession, packet.IPacket) error
	OnTick(ISession, time.Duration)
	OnDisconnect(ISession, error)
	OnError(error)
}

type ISessionEventHandlerEx interface {
	ISessionEventHandler
	OnChannel(any)
}
