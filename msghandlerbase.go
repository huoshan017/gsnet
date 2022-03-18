package gsnet

type MsgData struct {
}

// 基礎消息处理器
type MsgHandlerBase struct {
	msgDecoder IMsgDecoder
	handles    map[uint32]func(ISession, []byte) error
}

func NewMsgHandlerBase(msgDecoder IMsgDecoder) *MsgHandlerBase {
	h := &MsgHandlerBase{}
	h.init(msgDecoder)
	return h
}

func (h *MsgHandlerBase) init(msgDecoder IMsgDecoder) {
	h.msgDecoder = msgDecoder
	h.handles = make(map[uint32]func(ISession, []byte) error)
}

func (h *MsgHandlerBase) RegisterHandle(msgid uint32, handle func(ISession, []byte) error) {
	h.handles[msgid] = handle
}

func (h *MsgHandlerBase) OnMessage(sess ISession, msg MsgData) error {
	return nil
}

func (h *MsgHandlerBase) OnData(sess ISession, data []byte) error {
	msgid, msgdata := h.msgDecoder.Decode(data)
	handle, o := h.handles[msgid]
	if !o {
		e := ErrNoMsgHandleFunc(msgid)
		CheckAndRegisterNoDisconnectError(e)
		return e
	}
	return handle(sess, msgdata)
}

func (h *MsgHandlerBase) Send(sess ISession, msgid uint32, msgdata []byte) error {
	data := h.msgDecoder.Encode(msgid, msgdata)
	return sess.Send(data)
}

func NewDefaultMsgHandlerBase() *MsgHandlerBase {
	return NewMsgHandlerBase(&DefaultMsgDecoder{})
}
