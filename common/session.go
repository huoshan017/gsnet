package common

import (
	"github.com/huoshan017/gsnet/packet"
	"github.com/huoshan017/gsnet/pool"
)

const (
	DefaultPacketChannelLength = 32
)

type Session struct {
	conn             IConn
	senderWithResend ISenderWithResend
	id               uint64
	key              uint64
	dataMap          map[string]any
	chPak            chan IdWithPacket
	inboundHandles   map[int32]func(ISession, int32, packet.IPacket) error
	resendData       *ResendData
}

func NewSession(conn IConn, id uint64) *Session {
	return &Session{
		conn: conn,
		id:   id,
	}
}

func NewSessionWithResend(conn IConn, id uint64, resendData *ResendData) *Session {
	sender, _ := conn.(ISenderWithResend)
	return &Session{
		conn:             conn,
		senderWithResend: sender,
		id:               id,
		resendData:       resendData,
	}
}

func (s *Session) GetId() uint64 {
	return s.id
}

func (s *Session) GetKey() uint64 {
	return s.key
}

func (s *Session) Conn() IConn {
	return s.conn
}

func (s *Session) Send(data []byte, toCopy bool) error {
	if s.senderWithResend != nil {
		return s.senderWithResend.send(data, toCopy, s.resendData)
	}
	return s.conn.Send(packet.PacketNormalData, data, toCopy)
}

func (s *Session) SendBytesArray(bytesArray [][]byte, toCopy bool) error {
	if s.senderWithResend != nil {
		return s.senderWithResend.sendBytesArray(bytesArray, toCopy, s.resendData)
	}
	return s.conn.SendBytesArray(packet.PacketNormalData, bytesArray, toCopy)
}

func (s *Session) SendPoolBuffer(pBytes *[]byte) error {
	if s.senderWithResend != nil {
		return s.senderWithResend.sendPoolBuffer(pBytes, packet.MemoryManagementPoolUserManualFree, s.resendData)
	}
	return s.conn.SendPoolBuffer(packet.PacketNormalData, pBytes, packet.MemoryManagementPoolUserManualFree)
}

func (s *Session) SendPoolBufferArray(pBytesArray []*[]byte) error {
	if s.senderWithResend != nil {
		return s.senderWithResend.sendPoolBufferArray(pBytesArray, packet.MemoryManagementPoolUserManualFree, s.resendData)
	}
	return s.conn.SendPoolBufferArray(packet.PacketNormalData, pBytesArray, packet.MemoryManagementPoolUserManualFree)
}

func (s *Session) Close() error {
	s.chPak = nil
	if s.resendData != nil {
		s.resendData.Dispose()
	}
	return s.conn.Close()
}

func (s *Session) CloseWaitSecs(secs int) error {
	if s.resendData != nil {
		s.resendData.Dispose()
	}
	return s.conn.CloseWait(secs)
}

func (s *Session) IsClosed() bool {
	return s.conn.IsClosed()
}

func (s *Session) AddInboundHandle(id int32, handle func(ISession, int32, packet.IPacket) error) {
	if s.inboundHandles == nil {
		s.inboundHandles = make(map[int32]func(ISession, int32, packet.IPacket) error)
	}
	s.inboundHandles[id] = handle
}

func (s *Session) RemoveInboundHandle(id int32) {
	delete(s.inboundHandles, id)
}

func (s *Session) GetInboundHandle(id int32) func(ISession, int32, packet.IPacket) error {
	h, o := s.inboundHandles[id]
	if !o {
		return nil
	}
	return h
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

func (s *Session) GetResendData() *ResendData {
	return s.resendData
}

type SessionEx struct {
	*Session
}

func NewSessionNoId(conn IConn) *SessionEx {
	return &SessionEx{
		Session: NewSession(conn, 0),
	}
}

func NewSessionNoIdWithResend(conn IConn, resend *ResendData) *SessionEx {
	return &SessionEx{
		Session: NewSessionWithResend(conn, 0, resend),
	}
}

func (s *SessionEx) SetId(id uint64) {
	s.id = id
}

type AgentSession struct {
	agentSessionId uint32
	sess           ISession
}

func NewAgentSession(agentSessionId uint32, sess ISession) *AgentSession {
	return &AgentSession{
		agentSessionId: agentSessionId,
		sess:           sess,
	}
}

func (sc *AgentSession) Reset(agentSessionId uint32, sess ISession) {
	sc.agentSessionId = agentSessionId
	sc.sess = sess
}

func (sc *AgentSession) GetId() uint64 {
	return sc.sess.GetId()
}

func (sc *AgentSession) GetKey() uint64 {
	return sc.sess.GetKey()
}

//func (sc *AgentSession) Conn() IConn {
//	return sc.sess.Conn()
//}

func (sc *AgentSession) GetAgentId() uint32 {
	return sc.agentSessionId
}

func (as *AgentSession) send(data []byte) error {
	temp := make([]byte, 4)
	Uint32ToBuffer(as.agentSessionId, temp)
	return as.sess.SendBytesArray([][]byte{temp, data}, false)
}

func (as *AgentSession) sendOnCopy(data []byte) error {
	buffer := pool.GetBuffPool().Alloc(4 + int32(len(data)))
	Uint32ToBuffer(as.agentSessionId, (*buffer)[:4])
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
	Uint32ToBuffer(as.agentSessionId, temp)
	datas = append([][]byte{temp}, datas...)
	return as.sess.SendBytesArray(datas, isCopy)
}

func (as *AgentSession) SendPoolBuffer(buffer *[]byte) error {
	buf := pool.GetBuffPool().Alloc(4)
	Uint32ToBuffer(as.agentSessionId, (*buf)[:])
	return as.sess.SendPoolBufferArray([]*[]byte{buf, buffer})
}

func (as *AgentSession) SendPoolBufferArray(bufferArray []*[]byte) error {
	buf := pool.GetBuffPool().Alloc(4)
	Uint32ToBuffer(as.agentSessionId, (*buf)[:])
	bufferArray = append([]*[]byte{buf}, bufferArray...)
	return as.sess.SendPoolBufferArray(bufferArray)
}

func (as *AgentSession) Close() error {
	return as.sess.Close()
}

func (sc *AgentSession) CloseWaitSecs(secs int) error {
	return sc.sess.CloseWaitSecs(secs)
}

func (sc *AgentSession) IsClosed() bool {
	return sc.sess.IsClosed()
}

func (sc *AgentSession) AddInboundHandle(id int32, handle func(ISession, int32, packet.IPacket) error) {
	sc.sess.AddInboundHandle(id, handle)
}

func (sc *AgentSession) RemoveInboundHandle(id int32) {
	sc.sess.RemoveInboundHandle(id)
}

func (sc *AgentSession) GetInboundHandle(id int32) func(ISession, int32, packet.IPacket) error {
	return sc.sess.GetInboundHandle(id)
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
