package common

import "github.com/huoshan017/gsnet/common/packet"

type Session struct {
	conn       IConn
	id         uint64
	dataMap    map[string]interface{}
	resendData *ResendData
}

func NewSession(conn IConn, id uint64) *Session {
	return &Session{
		conn:    conn,
		id:      id,
		dataMap: make(map[string]interface{}),
	}
}

func NewSessionNoId(conn IConn) *Session {
	return &Session{
		conn: conn,
	}
}

func (s *Session) Send(data []byte, toCopy bool) error {
	return s.conn.Send(packet.PacketNormalData, data, toCopy)
}

func (s *Session) SendBytesArray(bytesArray [][]byte, toCopy bool) error {
	return s.conn.SendBytesArray(packet.PacketNormalData, bytesArray, toCopy)
}

func (s *Session) SendPoolBuffer(pBytes *[]byte, mmType packet.MemoryManagementType) error {
	return s.conn.SendPoolBuffer(packet.PacketNormalData, pBytes, mmType)
}

func (s *Session) SendPoolBufferArray(pBytesArray []*[]byte, mmType packet.MemoryManagementType) error {
	return s.conn.SendPoolBufferArray(packet.PacketNormalData, pBytesArray, mmType)
}

func (s *Session) Close() {
	s.conn.Close()
}

func (s *Session) CloseWaitSecs(secs int) {
	s.conn.CloseWait(secs)
}

func (s *Session) GetId() uint64 {
	return s.id
}

func (s *Session) SetData(k string, d interface{}) {
	s.dataMap[k] = d
}

func (s *Session) GetData(k string) interface{} {
	return s.dataMap[k]
}

func (s *Session) SetResendData(resendData *ResendData) {
	s.resendData = resendData
}

func (s *Session) GetResendData() *ResendData {
	return s.resendData
}
