package msg

import (
	"errors"
	"reflect"
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/common/packet"
	"github.com/huoshan017/gsnet/common/pool"
)

var (
	ErrMsgIdMapTypeNotFound = errors.New("gsnet: type map to message id not found")
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
	common.ISessionHandler
	connectHandle    func(common.ISession)
	disconnectHandle func(common.ISession, error)
	tickHandle       func(common.ISession, time.Duration)
	errHandle        func(error)
	handleMap        map[MsgIdType]func(common.ISession, interface{}) error
	codec            IMsgCodec
	mapper           *IdMsgMapper
}

func newMsgHandler(codec IMsgCodec, mapper *IdMsgMapper) *msgHandler {
	d := &msgHandler{}
	d.handleMap = make(map[MsgIdType]func(common.ISession, interface{}) error)
	d.codec = codec
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

func (d *msgHandler) RegisterHandle(msgid MsgIdType, handle func(common.ISession, interface{}) error) {
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
	//msgid, msgdata := d.msgDecoder.Decode(*pak.Data())
	var (
		msgid   MsgIdType
		msgdata []byte
	)
	data := *pak.Data()
	msgid, msgdata = splitIdAndMsg(data)
	msgobj := d.mapper.GetReflectNewObject(msgid)
	if msgobj == nil {
		return ErrMsgIdMapTypeNotFound
	}
	d.codec.Decode(msgdata, msgobj)
	h, o := d.handleMap[msgid]
	if !o {
		e := common.ErrNoMsgHandleFunc(uint32(msgid))
		common.CheckAndRegisterNoDisconnectError(e)
		return e
	}
	return h(s, msgobj)
}

func (d *msgHandler) SendMsg(s common.ISession, msgid MsgIdType, msgobj interface{}) error {
	//data := d.msgDecoder.Encode(msgid, msgdata)
	msgdata, err := d.codec.Encode(msgobj)
	if err != nil {
		return err
	}
	//data := make([]byte, 4+len(msgdata))
	pData := pool.GetBuffPool().Alloc(int32(4 + len(msgdata)))
	genMsgIdHeader(msgid, *pData)
	copy((*pData)[4:], msgdata[:])
	return s.SendPoolBuffer(pData, packet.MemoryManagementPoolUserManualFree)
}

func (d *msgHandler) SendMsgNoCopy(s common.ISession, msgid MsgIdType, msgobj interface{}) error {
	msgdata, err := d.codec.Encode(msgobj)
	if err != nil {
		return err
	}
	idHeader := make([]byte, 4)
	genMsgIdHeader(msgid, idHeader)
	return s.SendBytesArray([][]byte{idHeader, msgdata}, false)
}

func splitIdAndMsg(data []byte) (msgid MsgIdType, msgdata []byte) {
	for i := 0; i < 4; i++ {
		msgid += MsgIdType(data[i] << (8 * (4 - i - 1)))
	}
	msgdata = data[4:]
	return
}

func genMsgIdHeader(msgid MsgIdType, idHeader []byte) {
	for i := 0; i < 4; i++ {
		idHeader[i] = byte(msgid >> (8 * (4 - i - 1)))
	}
}

func init() {
	common.RegisterNoDisconnectError(ErrMsgIdMapTypeNotFound)
}
