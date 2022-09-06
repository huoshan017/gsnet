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
	Close() error
	CloseWait(int) error
	IsClosed() bool
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
	GetKey() uint64
	Send([]byte, bool) error
	SendBytesArray([][]byte, bool) error
	SendPoolBuffer(*[]byte) error
	SendPoolBufferArray([]*[]byte) error
	Close() error
	CloseWaitSecs(int) error
	IsClosed() bool
	AddInboundHandle(int32, func(ISession, int32, packet.IPacket) error)
	RemoveInboundHandle(int32)
	GetInboundHandle(int32) func(ISession, int32, packet.IPacket) error
	GetPacketChannel() chan IdWithPacket
	SetUserData(string, any)
	GetUserData(string) any
}

// 连接提取器
type IConnGetter interface {
	Conn() IConn
}

// 代理会话接口
type IAgentSession interface {
	ISession
	GetAgentId() uint32
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
