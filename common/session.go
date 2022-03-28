package common

type Session struct {
	conn    IConn
	id      uint64
	dataMap map[string]interface{}
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
	return s.conn.Send(data, toCopy)
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