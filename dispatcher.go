package gsnet

type ServiceMsgDispatcher struct {
	handleMap map[uint32]func(*Session, []byte) error
	msgProto  IMsgProto
}

func NewServiceMsgDispatcher(msgProto IMsgProto) *ServiceMsgDispatcher {
	return &ServiceMsgDispatcher{
		handleMap: make(map[uint32]func(*Session, []byte) error),
		msgProto:  msgProto,
	}
}

func (d *ServiceMsgDispatcher) RegisterHandle(msgid uint32, handle func(*Session, []byte) error) {
	d.handleMap[msgid] = handle
}

func (d *ServiceMsgDispatcher) OnData(s *Session, data []byte) error {
	msgid, msgdata := d.msgProto.Decode(data)
	h, o := d.handleMap[msgid]
	if !o {
		return ErrNoMsgHandle
	}
	return h(s, msgdata)
}

func (d *ServiceMsgDispatcher) SendMsg(s *Session, msgid uint32, msgdata []byte) error {
	data := d.msgProto.Encode(msgid, msgdata)
	return s.Send(data)
}

type ClientMsgDispatcher struct {
	handleMap map[uint32]func([]byte) error
	msgProto  IMsgProto
}

func NewClientMsgDispatcher(msgProto IMsgProto) *ClientMsgDispatcher {
	return &ClientMsgDispatcher{
		handleMap: make(map[uint32]func([]byte) error),
		msgProto:  msgProto,
	}
}

func (d *ClientMsgDispatcher) RegisterHandle(msgid uint32, handle func([]byte) error) {
	d.handleMap[msgid] = handle
}

func (d *ClientMsgDispatcher) OnData(data []byte) error {
	msgid, msgdata := d.msgProto.Decode(data)
	h, o := d.handleMap[msgid]
	if !o {
		return ErrNoMsgHandle
	}
	return h(msgdata)
}

func (d *ClientMsgDispatcher) Encode(msgid uint32, msgdata []byte) []byte {
	return d.msgProto.Encode(msgid, msgdata)
}
