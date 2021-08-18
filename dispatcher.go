package gsnet

type MsgDispatcher struct {
	handleMap map[uint32]func(*Session, []byte) error
	msgProto  IMsgProto
}

func NewMsgDispatcher(msgProto IMsgProto) *MsgDispatcher {
	return &MsgDispatcher{
		handleMap: make(map[uint32]func(*Session, []byte) error),
		msgProto:  msgProto,
	}
}

func (d *MsgDispatcher) RegisterHandle(msgid uint32, handle func(*Session, []byte) error) {
	d.handleMap[msgid] = handle
}

func (d *MsgDispatcher) HandleData(s *Session, data []byte) error {
	msgid, msgdata := d.msgProto.Decode(data)
	h, o := d.handleMap[msgid]
	if !o {
		return ErrNoMsgHandle
	}
	return h(s, msgdata)
}

func (d *MsgDispatcher) SendMsg(s *Session, msgid uint32, msgdata []byte) error {
	data := d.msgProto.Encode(msgid, msgdata)
	return s.Send(data)
}
