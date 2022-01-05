package gsnet

import "time"

type MsgDispatcher struct {
	ISessionHandler
	connectHandle    func(ISession)
	disconnectHandle func(ISession, error)
	tickHandle       func(ISession, time.Duration)
	errHandle        func(error)
	handleMap        map[uint32]func(ISession, []byte) error
	msgProto         IMsgProto
}

func NewMsgDispatcher(msgProto IMsgProto) *MsgDispatcher {
	d := &MsgDispatcher{}
	d.handleMap = make(map[uint32]func(ISession, []byte) error)
	d.msgProto = msgProto
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

func (d *MsgDispatcher) RegisterHandle(msgid uint32, handle func(ISession, []byte) error) {
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

func (d *MsgDispatcher) OnData(s ISession, data []byte) error {
	msgid, msgdata := d.msgProto.Decode(data)
	h, o := d.handleMap[msgid]
	if !o {
		e := ErrNoMsgHandleFunc(msgid)
		RegisterNoDisconnectError(e)
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
	data := d.msgProto.Encode(msgid, msgdata)
	return s.Send(data)
}
