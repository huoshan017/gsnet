package msg

import (
	"fmt"
	"reflect"
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/packet"
)

var (
	ErrMsgIdMapperTypeNotFound = func(msgid MsgIdType) error {
		return fmt.Errorf("gsnet: type map to message id %v not found", msgid)
	}
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

func (ma *IdMsgMapper) GetReflectNewObject(id MsgIdType) any {
	rt, o := ma.m[id]
	if !o {
		return nil
	}
	return reflect.New(rt.Elem()).Interface()
}

type msgHandlerCommon struct {
	connectHandle    func(*MsgSession)
	disconnectHandle func(*MsgSession, error)
	tickHandle       func(*MsgSession, time.Duration)
	errHandle        func(error)
	sess             *MsgSession
}

func newMsgHandlerCommon(codec IMsgCodec, mapper *IdMsgMapper) *msgHandlerCommon {
	d := &msgHandlerCommon{
		sess: &MsgSession{codec: codec, mapper: mapper},
	}
	return d
}

func (d *msgHandlerCommon) SetConnectHandle(handle func(*MsgSession)) {
	d.connectHandle = handle
}

func (d *msgHandlerCommon) SetDisconnectHandle(handle func(*MsgSession, error)) {
	d.disconnectHandle = handle
}

func (d *msgHandlerCommon) SetTickHandle(handle func(*MsgSession, time.Duration)) {
	d.tickHandle = handle
}

func (d *msgHandlerCommon) SetErrorHandle(handle func(error)) {
	d.errHandle = handle
}

func (d *msgHandlerCommon) OnConnect(s common.ISession) {
	if d.connectHandle != nil {
		d.sess.sess = s
		d.connectHandle(d.sess)
	}
}

func (d *msgHandlerCommon) OnDisconnect(s common.ISession, err error) {
	if d.disconnectHandle != nil {
		d.sess.sess = s
		d.disconnectHandle(d.sess, err)
	}
}

func (d *msgHandlerCommon) OnTick(s common.ISession, tick time.Duration) {
	if d.tickHandle != nil {
		d.sess.sess = s
		d.tickHandle(d.sess, tick)
	}
}

func (d *msgHandlerCommon) OnError(err error) {
	if d.errHandle != nil {
		d.errHandle(err)
	}
}

func (d *msgHandlerCommon) SendMsg(s common.ISession, msgid MsgIdType, msgobj any) error {
	d.sess.sess = s
	return d.sess.SendMsg(msgid, msgobj)
}

func (d *msgHandlerCommon) SendMsgNoCopy(s common.ISession, msgid MsgIdType, msgobj any) error {
	d.sess.sess = s
	return d.sess.SendMsgNoCopy(msgid, msgobj)
}

type msgHandlerClient struct {
	msgHandlerCommon
	handleMap map[MsgIdType]func(*MsgSession, any) error
}

func newMsgHandlerClient(codec IMsgCodec, mapper *IdMsgMapper) *msgHandlerClient {
	return &msgHandlerClient{
		msgHandlerCommon: *newMsgHandlerCommon(codec, mapper),
		handleMap:        make(map[MsgIdType]func(*MsgSession, any) error),
	}
}

func (d *msgHandlerClient) RegisterHandle(msgid MsgIdType, handle func(*MsgSession, any) error) {
	d.handleMap[msgid] = handle
}

func (d *msgHandlerClient) OnPacket(s common.ISession, pak packet.IPacket) error {
	msgid, msgobj, err := d.sess.splitIdAndMsg(pak.Data())
	if err != nil {
		return err
	}
	h, o := d.handleMap[msgid]
	if !o {
		e := common.ErrNoMsgHandle(uint32(msgid))
		common.CheckAndRegisterNoDisconnectError(e)
		return e
	}
	d.sess.sess = s
	return h(d.sess, msgobj)
}

type msgHandlerServer struct {
	msgHandlerCommon
	sessionHandler IMsgSessionEventHandler
	msgHandle      func(*MsgSession, MsgIdType, any) error
}

func newMsgHandlerServer(sessionHandler IMsgSessionEventHandler, codec IMsgCodec, mapper *IdMsgMapper) *msgHandlerServer {
	server := &msgHandlerServer{
		msgHandlerCommon: *newMsgHandlerCommon(codec, mapper),
		sessionHandler:   sessionHandler,
	}
	server.msgHandlerCommon.SetConnectHandle(sessionHandler.OnConnected)
	server.msgHandlerCommon.SetDisconnectHandle(sessionHandler.OnDisconnected)
	server.msgHandlerCommon.SetTickHandle(sessionHandler.OnTick)
	server.msgHandlerCommon.SetErrorHandle(sessionHandler.OnError)
	server.msgHandle = sessionHandler.OnMsgHandle
	return server
}

func (d *msgHandlerServer) OnPacket(s common.ISession, pak packet.IPacket) error {
	msgid, msgobj, err := d.sess.splitIdAndMsg(pak.Data())
	if err != nil {
		return err
	}
	d.sess.sess = s
	return d.msgHandle(d.sess, msgid, msgobj)
}
