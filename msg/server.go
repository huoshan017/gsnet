package msg

import (
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/server"
)

type ServerSessionHandles struct {
	connectedHandle    func(*MsgSession)
	disconnectedHandle func(*MsgSession, error)
	tickHandle         func(*MsgSession, time.Duration)
	errorHandle        func(error)
	msgHandles         map[MsgIdType]func(*MsgSession, interface{}) error
}

func CreateServerSessionHandles() *ServerSessionHandles {
	return &ServerSessionHandles{
		msgHandles: make(map[MsgIdType]func(*MsgSession, interface{}) error),
	}
}

func (h *ServerSessionHandles) SetConnectedHandle(handle func(*MsgSession)) {
	h.connectedHandle = handle
}

func (h *ServerSessionHandles) SetDisconnectedHandle(handle func(*MsgSession, error)) {
	h.disconnectedHandle = handle
}

func (h *ServerSessionHandles) SetTickHandle(handle func(*MsgSession, time.Duration)) {
	h.tickHandle = handle
}

func (h *ServerSessionHandles) SetErrorHandle(handle func(error)) {
	h.errorHandle = handle
}

func (h *ServerSessionHandles) SetMsgHandle(msgid MsgIdType, handle func(*MsgSession, interface{}) error) {
	h.msgHandles[msgid] = handle
}

func (h *ServerSessionHandles) SetMsgHandleList(list []struct {
	Id     MsgIdType
	Handle func(*MsgSession, interface{}) error
}) {
	for _, d := range list {
		h.msgHandles[d.Id] = d.Handle
	}
}

func newMsgHandlerWithServer(args ...interface{}) common.ISessionHandler {
	codec := args[0].(IMsgCodec)
	handles := args[1].(*ServerSessionHandles)
	mapper := args[2].(*IdMsgMapper)
	handler := newMsgHandler(codec, mapper)
	handler.SetConnectHandle(handles.connectedHandle)
	handler.SetDisconnectHandle(handles.disconnectedHandle)
	handler.SetTickHandle(handles.tickHandle)
	handler.SetErrorHandle(handles.errorHandle)
	for msgid, msghandle := range handles.msgHandles {
		handler.RegisterHandle(msgid, msghandle)
	}
	return handler
}

// MsgServer struct
type MsgServer struct {
	*server.Server
	created    bool
	options    []common.Option
	createArgs []interface{}
	handles    *ServerSessionHandles
}

// NewMsgServerDirectly create new message server directly
func NewMsgServerDirectly(msgCodec IMsgCodec, handles *ServerSessionHandles, idMsgMapper *IdMsgMapper, options ...common.Option) *MsgServer {
	funcArgs := []interface{}{msgCodec, handles, idMsgMapper}
	options = append(options, server.WithNewSessionHandlerFuncArgs(funcArgs...))
	s := &MsgServer{
		Server:  server.NewServer(newMsgHandlerWithServer, options...),
		created: true,
	}
	return s
}

// NewMsgServer create new message server
func NewMsgServer(msgCodec IMsgCodec, idMsgMapper *IdMsgMapper, options ...common.Option) *MsgServer {
	s := &MsgServer{createArgs: []interface{}{msgCodec, idMsgMapper}, options: options, handles: CreateServerSessionHandles()}
	return s
}

func (s *MsgServer) SetSessionHandles(handles struct {
	ConnectedHandle    func(*MsgSession)
	DisconnectedHandle func(*MsgSession, error)
	TickHandle         func(*MsgSession, time.Duration)
	ErrorHandle        func(error)
	MsgHandles         map[MsgIdType]func(*MsgSession, interface{}) error
}) {
	if s.created {
		return
	}
	s.handles.connectedHandle = handles.ConnectedHandle
	s.handles.disconnectedHandle = handles.DisconnectedHandle
	s.handles.tickHandle = handles.TickHandle
	s.handles.errorHandle = handles.ErrorHandle
	s.handles.msgHandles = handles.MsgHandles
}

func (s *MsgServer) SetSessionConnectedHandle(handle func(*MsgSession)) {
	if s.created {
		return
	}
	s.handles.connectedHandle = handle
}

func (s *MsgServer) SetSessionDisconnectedHandle(handle func(*MsgSession, error)) {
	if s.created {
		return
	}
	s.handles.disconnectedHandle = handle
}

func (s *MsgServer) SetSessionTickHandle(handle func(*MsgSession, time.Duration)) {
	if s.created {
		return
	}
	s.handles.tickHandle = handle
}

func (s *MsgServer) SetSessionErrorHandle(handle func(error)) {
	if s.created {
		return
	}
	s.handles.errorHandle = handle
}

func (s *MsgServer) SetMsgSessionHandle(msgid MsgIdType, handle func(*MsgSession, interface{}) error) {
	if s.created {
		return
	}
	s.handles.SetMsgHandle(msgid, handle)
}

func (s *MsgServer) SetMsgSessionHandleList(handleList []struct {
	Id     MsgIdType
	Handle func(*MsgSession, interface{}) error
}) {
	if s.created {
		return
	}
	for _, d := range handleList {
		s.handles.SetMsgHandle(d.Id, d.Handle)
	}
}

// MsgServer.Start start server
func (s *MsgServer) Listen(address string) error {
	if s.Server == nil {
		s.createArgs = append(s.createArgs[:1], append([]interface{}{s.handles}, s.createArgs[1:]...)...)
		s.options = append(s.options, server.WithNewSessionHandlerFuncArgs(s.createArgs...))
		s.Server = server.NewServer(newMsgHandlerWithServer, s.options...)
		s.created = true
	}
	return s.Server.Listen(address)
}

// NewPBMsgServer create a protobuf message server
func NewPBMsgServer(idMsgMapper *IdMsgMapper, options ...common.Option) *MsgServer {
	return NewMsgServer(&ProtobufCodec{}, idMsgMapper, options...)
}

// NewJsonMsgServer create a json message server
func NewJsonMsgServerr(idMsgMapper *IdMsgMapper, options ...common.Option) *MsgServer {
	return NewMsgServer(&JsonCodec{}, idMsgMapper, options...)
}

// NewGobMsgServer create a gob message server
func NewGobMsgServer(idMsgMapper *IdMsgMapper, options ...common.Option) *MsgServer {
	return NewMsgServer(&GobCodec{}, idMsgMapper, options...)
}

// NewPBMsgServer create a protobuf message server directly
func NewPBMsgServerDirectly(handles *ServerSessionHandles, idMsgMapper *IdMsgMapper, options ...common.Option) *MsgServer {
	return NewMsgServerDirectly(&ProtobufCodec{}, handles, idMsgMapper, options...)
}

// NewJsonMsgServer create a json message server directly
func NewJsonMsgServerDirectly(handles *ServerSessionHandles, idMsgMapper *IdMsgMapper, options ...common.Option) *MsgServer {
	return NewMsgServerDirectly(&JsonCodec{}, handles, idMsgMapper, options...)
}

// NewGobMsgServer create a gob message server directly
func NewGobMsgServerDirectly(handles *ServerSessionHandles, idMsgMapper *IdMsgMapper, options ...common.Option) *MsgServer {
	return NewMsgServerDirectly(&GobCodec{}, handles, idMsgMapper, options...)
}
