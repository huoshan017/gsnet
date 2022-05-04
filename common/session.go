package common

import (
	"github.com/huoshan017/gsnet/packet"
	"github.com/huoshan017/gsnet/pool"
)

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

func (s *Session) RemoveInboundHandle(id int32) {
	if s.inboundHandles != nil {
		delete(s.inboundHandles, id)
	}
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

type AgentSession struct {
	agentId uint32
	sess    ISession
}

func NewAgentSession(agentId uint32, sess ISession) *AgentSession {
	return &AgentSession{
		agentId: agentId,
		sess:    sess,
	}
}

func (sc *AgentSession) GetId() uint64 {
	return sc.sess.GetId()
}

func (sc *AgentSession) GetAgentId() uint32 {
	return sc.agentId
}

func (as *AgentSession) send(data []byte) error {
	temp := make([]byte, 4)
	Uint32ToBuffer(as.agentId, temp)
	return as.sess.SendBytesArray([][]byte{temp, data}, false)
}

func (as *AgentSession) sendOnCopy(data []byte) error {
	buffer := pool.GetBuffPool().Alloc(4 + int32(len(data)))
	Uint32ToBuffer(as.agentId, (*buffer)[:4])
	copy((*buffer)[4:], data)
	return as.sess.SendPoolBuffer(buffer)
}

func (as *AgentSession) Send(data []byte, isCopy bool) error {
	if !isCopy {
		return as.send(data)
	} else {
		return as.sendOnCopy(data)
	}
}

func (as *AgentSession) SendBytesArray(datas [][]byte, isCopy bool) error {
	temp := make([]byte, 4)
	Uint32ToBuffer(as.agentId, temp)
	datas = append([][]byte{temp}, datas...)
	return as.sess.SendBytesArray(datas, isCopy)
}

func (as *AgentSession) SendPoolBuffer(buffer *[]byte) error {
	buf := pool.GetBuffPool().Alloc(4)
	Uint32ToBuffer(as.agentId, (*buf)[:])
	return as.sess.SendPoolBufferArray([]*[]byte{buf, buffer})
}

func (as *AgentSession) SendPoolBufferArray(bufferArray []*[]byte) error {
	buf := pool.GetBuffPool().Alloc(4)
	Uint32ToBuffer(as.agentId, (*buf)[:])
	bufferArray = append([]*[]byte{buf}, bufferArray...)
	return as.sess.SendPoolBufferArray(bufferArray)
}

func (as *AgentSession) Close() {
	as.sess.Close()
}

func (sc *AgentSession) CloseWaitSecs(secs int) {
	sc.sess.CloseWaitSecs(secs)
}

func (sc *AgentSession) AddInboundHandle(id int32, handle func(ISession, packet.IPacket) error) {
	sc.sess.AddInboundHandle(id, handle)
}

func (sc *AgentSession) RemoveInboundHandle(id int32) {
	sc.sess.RemoveInboundHandle(id)
}

func (sc *AgentSession) GetInboundHandles() map[int32]func(ISession, packet.IPacket) error {
	return sc.sess.GetInboundHandles()
}

func (sc *AgentSession) GetPacketChannel() chan IdWithPacket {
	return sc.sess.GetPacketChannel()
}

func (sc *AgentSession) SetUserData(key string, data any) {
	sc.sess.SetUserData(key, data)
}

func (sc *AgentSession) GetUserData(key string) any {
	return sc.sess.GetUserData(key)
}
