package gsnet

// 消息服务
type MsgService struct {
	*Service
	dispatcher *ServiceMsgDispatcher
}

func NewMsgService(callback IServiceCallback, options ...Option) *MsgService {
	s := &MsgService{}
	s.Service = NewService(callback, s.dispatcher, options...)
	if s.options.MsgProto == nil {
		s.dispatcher = NewServiceMsgDispatcher(&DefaultMsgProto{})
	} else {
		s.dispatcher = NewServiceMsgDispatcher(s.options.MsgProto)
	}
	s.Service.handler = s.dispatcher
	return s
}

func (s *MsgService) RegisterHandle(msgid uint32, handle func(ISession, []byte) error) {
	s.dispatcher.RegisterHandle(msgid, handle)
}

func (s *MsgService) Send(sess ISession, msgid uint32, msgdata []byte) error {
	return s.dispatcher.SendMsg(sess, msgid, msgdata)
}
