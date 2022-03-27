package common

import "time"

type MsgDispatcher struct {
	ISessionHandler
	connectHandle    func(ISession)
	disconnectHandle func(ISession, error)
	tickHandle       func(ISession, time.Duration)
	errHandle        func(error)
	handleMap        map[uint32]func(ISession, interface{}) error
	msgDecoder       IMsgDecoder
}

func NewMsgDispatcher(msgDecoder IMsgDecoder) *MsgDispatcher {
	d := &MsgDispatcher{}
	d.handleMap = make(map[uint32]func(ISession, interface{}) error)
	d.msgDecoder = msgDecoder
	return d
}

func (d *MsgDispatcher) SetConnectHandle(handle func(ISession)) {
	d.connectHandle = handle
}

func (d *MsgDispatcher) SetDisconnectHandle(handle func(ISession, error)) {
	d.disconnectHandle = handle
}

func (d *MsgDispatcher) SetTickHandle(handle func(ISession, time.Duration)) {
	d.tickHandle = handle
}

func (d *MsgDispatcher) SetErrorHandle(handle func(error)) {
	d.errHandle = handle
}

func (d *MsgDispatcher) RegisterHandle(msgid uint32, handle func(ISession, interface{}) error) {
	d.handleMap[msgid] = handle
}

func (d *MsgDispatcher) OnConnect(s ISession) {
	if d.connectHandle != nil {
		d.connectHandle(s)
	}
}

func (d *MsgDispatcher) OnDisconnect(s ISession, err error) {
	if d.disconnectHandle != nil {
		d.disconnectHandle(s, err)
	}
}

func (d *MsgDispatcher) OnTick(s ISession, tick time.Duration) {
	if d.tickHandle != nil {
		d.tickHandle(s, tick)
	}
}

func (d *MsgDispatcher) OnData(s ISession, data interface{}) error {
	dd, o := data.([]byte)
	if !o {
		panic("gsnet: data type must be []byte")
	}
	msgid, msgdata := d.msgDecoder.Decode(dd)
	h, o := d.handleMap[msgid]
	if !o {
		e := ErrNoMsgHandleFunc(msgid)
		CheckAndRegisterNoDisconnectError(e)
		return e
	}
	return h(s, msgdata)
}

func (d *MsgDispatcher) OnError(err error) {
	if d.errHandle != nil {
		d.errHandle(err)
	}
}

func (d *MsgDispatcher) SendMsg(s ISession, msgid uint32, msgdata []byte) error {
	data := d.msgDecoder.Encode(msgid, msgdata)
	return s.Send(data)
}
