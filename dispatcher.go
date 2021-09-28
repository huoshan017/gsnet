package gsnet

type MsgDispatcher struct {
	handleMap map[uint32]func(ISession, []byte) error
	msgProto  IMsgProto
}

func NewMsgDispatcher(msgProto IMsgProto) *MsgDispatcher {
	return &MsgDispatcher{
		handleMap: make(map[uint32]func(ISession, []byte) error),
		msgProto:  msgProto,
	}
}

func (d *MsgDispatcher) RegisterHandle(msgid uint32, handle func(ISession, []byte) error) {
	d.handleMap[msgid] = handle
}

func (d *MsgDispatcher) OnData(s ISession, data []byte) error {
	msgid, msgdata := d.msgProto.Decode(data)
	h, o := d.handleMap[msgid]
	if !o {
		return ErrNoMsgHandle
	}
	return h(s, msgdata)
}

func (d *MsgDispatcher) SendMsg(s ISession, msgid uint32, msgdata []byte) error {
	data := d.msgProto.Encode(msgid, msgdata)
	return s.Send(data)
}
