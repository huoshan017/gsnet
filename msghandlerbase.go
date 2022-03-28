package gsnet

import (
	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/common/packet"
)

type MsgData struct {
}

// 基礎消息处理器
type MsgHandlerBase struct {
	msgDecoder common.IMsgDecoder
	handles    map[uint32]func(common.ISession, packet.IPacket) error
}

func NewMsgHandlerBase(msgDecoder common.IMsgDecoder) *MsgHandlerBase {
	h := &MsgHandlerBase{}
	h.init(msgDecoder)
	return h
}

func (h *MsgHandlerBase) init(msgDecoder common.IMsgDecoder) {
	h.msgDecoder = msgDecoder
	h.handles = make(map[uint32]func(common.ISession, packet.IPacket) error)
}

func (h *MsgHandlerBase) RegisterHandle(msgid uint32, handle func(common.ISession, packet.IPacket) error) {
	h.handles[msgid] = handle
}

func (h *MsgHandlerBase) OnMessage(sess common.ISession, msg MsgData) error {
	return nil
}

func (h *MsgHandlerBase) OnPacket(sess common.ISession, p packet.IPacket) error {
	msgid, msgdata := h.msgDecoder.Decode(*p.Data())
	handle, o := h.handles[msgid]
	if !o {
		e := common.ErrNoMsgHandleFunc(msgid)
		common.CheckAndRegisterNoDisconnectError(e)
		return e
	}
	pak := packet.BytesPacket(msgdata)
	return handle(sess, &pak)
}

func (h *MsgHandlerBase) Send(sess common.ISession, msgid uint32, msgdata []byte) error {
	data := h.msgDecoder.Encode(msgid, msgdata)
	return sess.Send(data, false)
}

func NewDefaultMsgHandlerBase() *MsgHandlerBase {
	return NewMsgHandlerBase(&common.DefaultMsgDecoder{})
}
