package gsnet

type Session struct {
	conn IConn
	id   uint64
	data interface{}
}

func NewSession(conn IConn, id uint64) *Session {
	return &Session{
		conn: conn,
		id:   id,
	}
}

func (s *Session) Send(data []byte) error {
	return s.conn.Send(data)
}

func (s *Session) Close() {
	s.conn.Close()
}

func (s *Session) GetId() uint64 {
	return s.id
}

func (s *Session) SetData(d interface{}) {
	s.data = d
}

func (s *Session) GetData() interface{} {
	return s.data
}