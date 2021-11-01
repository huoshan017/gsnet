package gsnet

// 消息处理器
type MsgHandler struct {
	ISessionHandler
	msgProto IMsgProto
	handles  map[uint32]func(ISession, []byte) error
}

func NewMsgHandler(msgProto IMsgProto) *MsgHandler {
	h := &MsgHandler{}
	if msgProto == nil {
		msgProto = &DefaultMsgProto{}
	}
	h.init(msgProto)
	return h
}

func (h *MsgHandler) init(msgProto IMsgProto) {
	h.msgProto = msgProto
	h.handles = make(map[uint32]func(ISession, []byte) error)
}

func (h *MsgHandler) RegisterHandle(msgid uint32, handle func(ISession, []byte) error) {
	h.handles[msgid] = handle
}

func (h *MsgHandler) OnData(sess ISession, data []byte) error {
	msgid, msgdata := h.msgProto.Decode(data)
	handle, o := h.handles[msgid]
	if !o {
		return ErrNoMsgHandle
	}
	return handle(sess, msgdata)
}

func (h *MsgHandler) Send(sess ISession, msgid uint32, msgdata []byte) error {
	data := h.msgProto.Encode(msgid, msgdata)
	return sess.Send(data)
}
