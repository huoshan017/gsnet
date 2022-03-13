package gsnet

// 基礎消息处理器
type MsgHandlerBase struct {
	msgProto IMsgProto
	handles  map[uint32]func(ISession, []byte) error
}

func NewMsgHandlerBase(msgProto IMsgProto) *MsgHandlerBase {
	h := &MsgHandlerBase{}
	if msgProto == nil {
		msgProto = &DefaultMsgProto{}
	}
	h.init(msgProto)
	return h
}

func (h *MsgHandlerBase) init(msgProto IMsgProto) {
	h.msgProto = msgProto
	h.handles = make(map[uint32]func(ISession, []byte) error)
}

func (h *MsgHandlerBase) RegisterHandle(msgid uint32, handle func(ISession, []byte) error) {
	h.handles[msgid] = handle
}

func (h *MsgHandlerBase) OnData(sess ISession, data []byte) error {
	msgid, msgdata := h.msgProto.Decode(data)
	handle, o := h.handles[msgid]
	if !o {
		e := ErrNoMsgHandleFunc(msgid)
		RegisterNoDisconnectError(e)
		return e
	}
	return handle(sess, msgdata)
}

func (h *MsgHandlerBase) Send(sess ISession, msgid uint32, msgdata []byte) error {
	data := h.msgProto.Encode(msgid, msgdata)
	return sess.Send(data)
}
