package gsnet

import "time"

// 消息服务
type MsgService struct {
	*Service
	dispatcher *MsgDispatcher
}

func NewMsgService(options ...Option) *MsgService {
	s := &MsgService{}
	s.Service = NewService(s.dispatcher, options...)
	if s.options.MsgProto == nil {
		s.dispatcher = NewMsgDispatcher(&DefaultMsgProto{})
	} else {
		s.dispatcher = NewMsgDispatcher(s.options.MsgProto)
	}
	s.Service.handler = s.dispatcher
	return s
}

func (s *MsgService) SetConnectHandle(handle func(ISession)) {
	s.dispatcher.SetConnectHandle(handle)
}

func (s *MsgService) SetDisconnectHandle(handle func(ISession, error)) {
	s.dispatcher.SetDisconnectHandle(handle)
}

func (s *MsgService) SetTickHandle(handle func(ISession, time.Duration)) {
	s.dispatcher.SetTickHandle(handle)
}

func (s *MsgService) SetErrorHandle(handle func(error)) {
	s.dispatcher.SetErrorHandle(handle)
}

func (s *MsgService) RegisterHandle(msgid uint32, handle func(ISession, []byte) error) {
	s.dispatcher.RegisterHandle(msgid, handle)
}

func (s *MsgService) Send(sess ISession, msgid uint32, msgdata []byte) error {
	return s.dispatcher.SendMsg(sess, msgid, msgdata)
}
