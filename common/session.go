package common

import "github.com/huoshan017/gsnet/packet"

const (
	DefaultPacketChannelLength = 32
)

type Session struct {
	conn           IConn
	id             uint64
	dataMap        map[string]any
	chPak          chan IdWithPacket
	inboundHandles map[int32]func(ISession, packet.IPacket) error
	resendData     *ResendData
}

func NewSession(conn IConn, id uint64) *Session {
	return &Session{
		conn: conn,
		id:   id,
	}
}

func NewSessionNoId(conn IConn) *Session {
	return &Session{
		conn: conn,
	}
}

func (s *Session) GetId() uint64 {
	return s.id
}

func (s *Session) Send(data []byte, toCopy bool) error {
	return s.conn.Send(packet.PacketNormalData, data, toCopy)
}

func (s *Session) SendBytesArray(bytesArray [][]byte, toCopy bool) error {
	return s.conn.SendBytesArray(packet.PacketNormalData, bytesArray, toCopy)
}

func (s *Session) SendPoolBuffer(pBytes *[]byte) error {
	return s.conn.SendPoolBuffer(packet.PacketNormalData, pBytes, packet.MemoryManagementPoolUserManualFree)
}

func (s *Session) SendPoolBufferArray(pBytesArray []*[]byte) error {
	return s.conn.SendPoolBufferArray(packet.PacketNormalData, pBytesArray, packet.MemoryManagementPoolUserManualFree)
}

func (s *Session) Close() {
	if s.resendData != nil {
		s.resendData.Dispose()
	}
	s.conn.Close()
}

func (s *Session) CloseWaitSecs(secs int) {
	if s.resendData != nil {
		s.resendData.Dispose()
	}
	s.conn.CloseWait(secs)
}

func (s *Session) AddInboundHandle(id int32, handle func(ISession, packet.IPacket) error) {
	if s.inboundHandles == nil {
		s.inboundHandles = make(map[int32]func(ISession, packet.IPacket) error)
	}
	s.inboundHandles[id] = handle
}

func (s *Session) GetInboundHandles() map[int32]func(ISession, packet.IPacket) error {
	return s.inboundHandles
}

func (s *Session) GetPacketChannel() chan IdWithPacket {
	if s.chPak == nil {
		s.chPak = make(chan IdWithPacket, DefaultPacketChannelLength)
	}
	return s.chPak
}

func (s *Session) SetUserData(k string, d any) {
	if s.dataMap == nil {
		s.dataMap = make(map[string]any)
	}
	s.dataMap[k] = d
}

func (s *Session) GetUserData(k string) any {
	if s.dataMap == nil {
		return nil
	}
	return s.dataMap[k]
}

func (s *Session) SetResendData(resendData *ResendData) {
	s.resendData = resendData
}

func (s *Session) GetResendData() *ResendData {
	return s.resendData
}
