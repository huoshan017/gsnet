package gsnet

// 消息服务
type MsgService struct {
	*Service
	dispatcher *MsgDispatcher
}

func NewMsgService(callback IServiceCallback, options ...Option) *MsgService {
	s := &MsgService{}
	if s.options.MsgProto == nil {
		s.dispatcher = NewMsgDispatcher(&DefaultMsgProto{})
	} else {
		s.dispatcher = NewMsgDispatcher(s.options.MsgProto)
	}
	s.Service = NewService(callback, s.dispatcher, options...)
	return s
}

func (s *MsgService) RegisterHandle(msgid uint32, handle func(*Session, []byte) error) {
	s.dispatcher.RegisterHandle(msgid, handle)
}

func (s *MsgService) Send(sess *Session, msgid uint32, msgdata []byte) error {
	return s.dispatcher.SendMsg(sess, msgid, msgdata)
}
