package msg

import (
	"time"

	"github.com/huoshan017/gsnet/client"
	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/msg/codec"
	"github.com/huoshan017/gsnet/packet"
	"github.com/huoshan017/gsnet/server"
)

// struct AgentMsgClient
type MsgAgentClient struct {
	c       *client.AgentClient
	options MsgClientOptions
	codec   IMsgCodec
	mapper  *IdMsgMapper
}

// NewMsgAgentClient  create new agent message client
func NewMsgAgentClient(codec IMsgCodec, mapper *IdMsgMapper, options ...common.Option) *MsgAgentClient {
	return &MsgAgentClient{
		c:      client.NewAgentClient(options...),
		codec:  codec,
		mapper: mapper,
	}
}

// MsgAgentClient.Dial  dial to address
func (c *MsgAgentClient) Dial(address string) error {
	return c.c.Dial(address)
}

// MsgAgentClient.DialTimeout  dial to address with timeout
func (c *MsgAgentClient) DialTimeout(address string, timeout time.Duration) error {
	return c.c.DialTimeout(address, timeout)
}

// MsgAgentClient.SetConnectHandle  set connect handle
func (c *MsgAgentClient) SetConnectHandle(handle func(*MsgSession)) {
	var h func(common.ISession) = func(sess common.ISession) {
		ms := MsgSession{sess, c.codec, c.mapper, &c.options.MsgOptions}
		handle(&ms)
	}
	c.c.SetConnectHandle(h)
}

// MsgAgentClient.SetDisconnectHandle  set disconnect handle
func (c *MsgAgentClient) SetDisconnectHandle(handle func(*MsgSession, error)) {
	var h func(common.ISession, error) = func(sess common.ISession, err error) {
		ms := MsgSession{sess, c.codec, c.mapper, &c.options.MsgOptions}
		handle(&ms, err)
	}
	c.c.SetDisconnectHandle(h)
}

// MsgAgentClient.SetTickHandle  set tick handle
func (c *MsgAgentClient) SetTickHandle(handle func(*MsgSession, time.Duration)) {
	var h func(common.ISession, time.Duration) = func(sess common.ISession, tick time.Duration) {
		ms := MsgSession{sess, c.codec, c.mapper, &c.options.MsgOptions}
		handle(&ms, tick)
	}
	c.c.SetTickHandle(h)
}

// MsgAgentClient.SetErrorHandle  set error handle
func (c *MsgAgentClient) SetErrorHandle(handle func(error)) {
	c.c.SetErrorHandle(handle)
}

// MsgAgentClient.BoundHandleAndGetAgentSession  bound message handle and create message agent session
func (c *MsgAgentClient) BoundHandleAndGetAgentSession(sess *MsgSession, handle func(*MsgSession, MsgIdType, any) error) *MsgAgentSession {
	var h func(common.ISession, packet.IPacket) error = func(sess common.ISession, pak packet.IPacket) error {
		var ms = MsgSession{sess: sess, codec: c.codec, mapper: c.mapper, options: &c.options.MsgOptions}
		msgid, msgobj, err := ms.splitIdAndMsg(pak.Data())
		if err != nil {
			return err
		}
		return handle(&ms, msgid, msgobj)
	}
	agentSess := c.c.BoundHandleAndGetAgentSession(sess.GetSess(), h)
	return &MsgAgentSession{AgentSession: agentSess, c: c}
}

// struct AgentMsgSession
type MsgAgentSession struct {
	*common.AgentSession
	c *MsgAgentClient
}

// AgentMsgSession.Send
func (c *MsgAgentSession) Send(msgId MsgIdType, msgObj any) error {
	var ms = MsgSession{codec: c.c.codec, mapper: c.c.mapper, options: &c.c.options.MsgOptions}
	dataArray, err := ms.serializeMsg(msgId, msgObj)
	if err != nil {
		return err
	}
	return c.AgentSession.SendBytesArray(dataArray, false)
}

// AgentMsgSession.SendOnCopy
func (c *MsgAgentSession) SendOnCopy(msgId MsgIdType, msgObj any) error {
	var ms = MsgSession{codec: c.c.codec, mapper: c.c.mapper, options: &c.c.options.MsgOptions}
	pData, err := ms.serializeMsgOnCopy(msgId, msgObj)
	if err != nil {
		return err
	}
	return c.AgentSession.SendPoolBuffer(pData)
}

// NewProtobufMsgAgentClient create protobuf message agent client
func NewProtobufMsgAgentClient(idMsgMapper *IdMsgMapper, options ...common.Option) *MsgAgentClient {
	return NewMsgAgentClient(&codec.ProtobufCodec{}, idMsgMapper, options...)
}

// NewJsonMsgAgentClient create json message agent client
func NewJsonMsgAgentClient(idMsgMapper *IdMsgMapper, options ...common.Option) *MsgAgentClient {
	return NewMsgAgentClient(&codec.JsonCodec{}, idMsgMapper, options...)
}

// NewGobMsgAgentClient create gob message agent client
func NewGobMsgAgentClient(idMsgMapper *IdMsgMapper, options ...common.Option) *MsgAgentClient {
	return NewMsgAgentClient(&codec.GobCodec{}, idMsgMapper, options...)
}

// NewThriftMsgAgentClient create thrift message agent client
func NewThriftMsgAgentClient(idMsgMapper *IdMsgMapper, options ...common.Option) *MsgAgentClient {
	return NewMsgAgentClient(&codec.ThriftCodec{}, idMsgMapper, options...)
}

// NewMsgpackMsgAgentClient create msgpack message agent client
func NewMsgpackMsgAgentClient(idMsgMapper *IdMsgMapper, options ...common.Option) *MsgAgentClient {
	return NewMsgAgentClient(&codec.MsgpackCodec{}, idMsgMapper, options...)
}

// struct AgentMsgServer
type MsgAgentServer struct {
	server  *server.AgentServer
	options MsgServerOptions
}

// NewMsgAgentServer create new message agent server
func NewMsgAgentServer(newFunc NewMsgSessionHandlerFunc, funcArgs []any, codec IMsgCodec, mapper *IdMsgMapper, options ...common.Option) *MsgAgentServer {
	s := &MsgAgentServer{}
	newSessionHandlerFunc := prepareMsgServer(newFunc, funcArgs, codec, mapper, &s.options, options...)
	s.server = server.NewAgentServerWithOptions(newSessionHandlerFunc, &s.options.ServerOptions)
	return s
}

// MsgAgentServer.Listen
func (s *MsgAgentServer) Listen(address string) error {
	return s.server.Listen(address)
}

// MsgAgentServer.Start
func (s *MsgAgentServer) Start() {
	s.server.Start()
}

// MsgAgentServer.ListenAndServe
func (s *MsgAgentServer) ListenAndServe(address string) error {
	return s.server.ListenAndServe(address)
}

// MsgAgentServer.End
func (s *MsgAgentServer) End() {
	s.server.End()
}

// NewProtobufMsgAgentServer create a protobuf message agent server
func NewProtobufMsgAgentServer(newFunc NewMsgSessionHandlerFunc, funcArgs []any, idMsgMapper *IdMsgMapper, options ...common.Option) *MsgAgentServer {
	return NewMsgAgentServer(newFunc, funcArgs, &codec.ProtobufCodec{}, idMsgMapper, options...)
}

// NewJsonMsgAgentServer create a json message agent server
func NewJsonMsgAgentServerr(newFunc NewMsgSessionHandlerFunc, funcArgs []any, idMsgMapper *IdMsgMapper, options ...common.Option) *MsgAgentServer {
	return NewMsgAgentServer(newFunc, funcArgs, &codec.JsonCodec{}, idMsgMapper, options...)
}

// NewGobMsgAgentServer create a gob message agent server
func NewGobMsgAgentServer(newFunc NewMsgSessionHandlerFunc, funcArgs []any, idMsgMapper *IdMsgMapper, options ...common.Option) *MsgAgentServer {
	return NewMsgAgentServer(newFunc, funcArgs, &codec.GobCodec{}, idMsgMapper, options...)
}

// NewThriftMsgAgentServer create a thrift message agent server
func NewThriftMsgAgentServer(newFunc NewMsgSessionHandlerFunc, funcArgs []any, idMsgMapper *IdMsgMapper, options ...common.Option) *MsgAgentServer {
	return NewMsgAgentServer(newFunc, funcArgs, &codec.ThriftCodec{}, idMsgMapper, options...)
}

// NewMsgpackMsgAgentServer create a msgpack message agent server
func NewMsgpackMsgAgentServer(newFunc NewMsgSessionHandlerFunc, funcArgs []any, idMsgMapper *IdMsgMapper, options ...common.Option) *MsgAgentServer {
	return NewMsgAgentServer(newFunc, funcArgs, &codec.MsgpackCodec{}, idMsgMapper, options...)
}
