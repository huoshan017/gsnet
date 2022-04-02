package msg

import (
	"errors"
	"reflect"
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/common/packet"
)

var (
	ErrMsgIdMapperTypeNotFound = errors.New("gsnet: type map to message id not found")
)

/*
// 基礎消息处理器
type MsgHandlerBase struct {
	msgDecoder IMsgDecoder
	handles    map[uint32]func(common.ISession, packet.IPacket) error
}

func NewMsgHandlerBase(msgDecoder IMsgDecoder) *MsgHandlerBase {
	h := &MsgHandlerBase{}
	h.init(msgDecoder)
	return h
}

func (h *MsgHandlerBase) init(msgDecoder IMsgDecoder) {
	h.msgDecoder = msgDecoder
	h.handles = make(map[uint32]func(common.ISession, packet.IPacket) error)
}

func (h *MsgHandlerBase) RegisterHandle(msgid uint32, handle func(common.ISession, packet.IPacket) error) {
	h.handles[msgid] = handle
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
	return NewMsgHandlerBase(&DefaultMsgDecoder{})
}
*/

type IdMsgMapper struct {
	m map[MsgIdType]reflect.Type
}

func CreateIdMsgMapper() *IdMsgMapper {
	return &IdMsgMapper{
		m: make(map[MsgIdType]reflect.Type),
	}
}

func CreateIdMsgMapperWith(m map[MsgIdType]reflect.Type) *IdMsgMapper {
	return &IdMsgMapper{
		m: m,
	}
}

func (ma *IdMsgMapper) AddMap(id MsgIdType, typ reflect.Type) {
	ma.m[id] = typ
}

func (ma *IdMsgMapper) GetReflectNewObject(id MsgIdType) interface{} {
	rt, o := ma.m[id]
	if !o {
		return nil
	}
	return reflect.New(rt.Elem()).Interface()
}

type msgHandler struct {
	connectHandle    func(common.ISession)
	disconnectHandle func(common.ISession, error)
	tickHandle       func(common.ISession, time.Duration)
	errHandle        func(error)
	handleMap        map[MsgIdType]func(*MsgSession, interface{}) error
	sess             *MsgSession
}

func newMsgHandler(codec IMsgCodec, mapper *IdMsgMapper) *msgHandler {
	d := &msgHandler{
		handleMap: make(map[MsgIdType]func(*MsgSession, interface{}) error),
		sess:      &MsgSession{codec: codec, mapper: mapper},
	}
	return d
}

func (d *msgHandler) SetConnectHandle(handle func(common.ISession)) {
	d.connectHandle = handle
}

func (d *msgHandler) SetDisconnectHandle(handle func(common.ISession, error)) {
	d.disconnectHandle = handle
}

func (d *msgHandler) SetTickHandle(handle func(common.ISession, time.Duration)) {
	d.tickHandle = handle
}

func (d *msgHandler) SetErrorHandle(handle func(error)) {
	d.errHandle = handle
}

func (d *msgHandler) RegisterHandle(msgid MsgIdType, handle func(*MsgSession, interface{}) error) {
	d.handleMap[msgid] = handle
}

func (d *msgHandler) OnConnect(s common.ISession) {
	if d.connectHandle != nil {
		d.connectHandle(s)
	}
}

func (d *msgHandler) OnDisconnect(s common.ISession, err error) {
	if d.disconnectHandle != nil {
		d.disconnectHandle(s, err)
	}
}

func (d *msgHandler) OnTick(s common.ISession, tick time.Duration) {
	if d.tickHandle != nil {
		d.tickHandle(s, tick)
	}
}

func (d *msgHandler) OnError(err error) {
	if d.errHandle != nil {
		d.errHandle(err)
	}
}

func (d *msgHandler) OnPacket(s common.ISession, pak packet.IPacket) error {
	msgid, msgobj, err := d.sess.splitIdAndMsg(*pak.Data())
	if err != nil {
		return err
	}
	h, o := d.handleMap[msgid]
	if !o {
		e := common.ErrNoMsgHandleFunc(uint32(msgid))
		common.CheckAndRegisterNoDisconnectError(e)
		return e
	}
	d.sess.sess = s
	return h(d.sess, msgobj)
}

func (d *msgHandler) SendMsg(s common.ISession, msgid MsgIdType, msgobj interface{}) error {
	d.sess.sess = s
	return d.sess.SendMsg(msgid, msgobj)
}

func (d *msgHandler) SendMsgNoCopy(s common.ISession, msgid MsgIdType, msgobj interface{}) error {
	d.sess.sess = s
	return d.sess.SendMsgNoCopy(msgid, msgobj)
}

func init() {
	common.RegisterNoDisconnectError(ErrMsgIdMapperTypeNotFound)
}
