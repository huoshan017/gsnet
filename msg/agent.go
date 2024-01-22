package msg

import (
	"time"

	"github.com/huoshan017/gsnet/client"
	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/msg/codec"
	"github.com/huoshan017/gsnet/options"
	"github.com/huoshan017/gsnet/packet"
	"github.com/huoshan017/gsnet/server"
)

// struct AgentMsg
type MsgAgent struct {
	c       *client.Agent
	options MsgClientOptions
	codec   IMsgCodec
	mapper  *IdMsgMapper
}

// NewMsgAgent  create new agent message client
func NewMsgAgent(codec IMsgCodec, mapper *IdMsgMapper, options ...options.Option) *MsgAgent {
	return &MsgAgent{
		c:      client.NewAgent(options...),
		codec:  codec,
		mapper: mapper,
	}
}

// MsgAgent.Dial  dial to address
func (c *MsgAgent) Dial(address string) error {
	return c.c.Dial(address)
}

// MsgAgent.DialTimeout  dial to address with timeout
func (c *MsgAgent) DialTimeout(address string, timeout time.Duration) error {
	return c.c.DialTimeout(address, timeout)
}

// MsgAgent.DialAsync  dial to address async
func (c *MsgAgent) DialAsync(address string, timeout time.Duration, callback func(error)) {
	c.c.DialAsync(address, timeout, callback)
}

// MsgAgent.Close
func (c *MsgAgent) Close() {
	c.c.Close()
}

// MsgAgent.CloseWait
func (c *MsgAgent) CloseWait(secs int) {
	c.c.CloseWait(secs)
}

// MsgAgent.IsNotConnect
func (c *MsgAgent) IsNotConnect() bool {
	return c.c.IsNotConnect()
}

// MsgAgent.IsConnecting
func (c *MsgAgent) IsConnecting() bool {
	return c.c.IsConnecting()
}

// MsgAgent.IsConnected
func (c *MsgAgent) IsConnected() bool {
	return c.c.IsConnected()
}

// MsgAgent.IsReady
func (c *MsgAgent) IsReady() bool {
	return c.c.IsReady()
}

// MsgAgent.IsDisconnecting
func (c *MsgAgent) IsDisconnecting() bool {
	return c.c.IsDisconnecting()
}

// MsgAgent.IsDisconnected
func (c *MsgAgent) IsDisconnected() bool {
	return c.c.IsDisconnected()
}

// MsgAgent.SetConnectHandle  set connect handle
func (c *MsgAgent) SetConnectHandle(handle func(*MsgSession)) {
	var h func(common.ISession) = func(sess common.ISession) {
		ms := MsgSession{sess, c.codec, c.mapper, &c.options.MsgOptions}
		handle(&ms)
	}
	c.c.SetConnectHandle(h)
}

func (c *MsgAgent) SetReadyHandle(handle func(*MsgSession)) {
	var h func(common.ISession) = func(sess common.ISession) {
		ms := MsgSession{sess, c.codec, c.mapper, &c.options.MsgOptions}
		handle(&ms)
	}
	c.c.SetReadyHandle(h)
}

// MsgAgent.SetDisconnectHandle  set disconnect handle
func (c *MsgAgent) SetDisconnectHandle(handle func(*MsgSession, error)) {
	var h func(common.ISession, error) = func(sess common.ISession, err error) {
		ms := MsgSession{sess, c.codec, c.mapper, &c.options.MsgOptions}
		handle(&ms, err)
	}
	c.c.SetDisconnectHandle(h)
}

// MsgAgent.SetTickHandle  set tick handle
func (c *MsgAgent) SetTickHandle(handle func(*MsgSession, time.Duration)) {
	var h func(common.ISession, time.Duration) = func(sess common.ISession, tick time.Duration) {
		ms := MsgSession{sess, c.codec, c.mapper, &c.options.MsgOptions}
		handle(&ms, tick)
	}
	c.c.SetTickHandle(h)
}

// MsgAgent.SetErrorHandle  set error handle
func (c *MsgAgent) SetErrorHandle(handle func(error)) {
	c.c.SetErrorHandle(handle)
}

// MsgAgent.BoundSession  bound message handle and create message agent session
func (c *MsgAgent) BoundServerSession(sess *MsgSession, handle func(*MsgSession, int32, MsgIdType, any) error) *MsgAgentSession {
	var h = func(sess common.ISession, agentId int32, pak packet.IPacket) error {
		var ms = MsgSession{sess: sess, codec: c.codec, mapper: c.mapper, options: &c.options.MsgOptions}
		msgid, msgobj, err := ms.splitIdAndMsg(pak.Data())
		if err != nil {
			return err
		}
		return handle(&ms, agentId, msgid, msgobj)
	}
	agentSess := c.c.BoundServerSession(sess.GetSess(), h)
	return &MsgAgentSession{AgentSession: agentSess, c: c}
}

// MsgAgent.UnboundSession
func (c *MsgAgent) UnboundServerSession(sess *MsgSession, asess *MsgAgentSession) {
	c.c.UnboundServerSession(sess.GetSess(), asess.AgentSession)
}

// struct AgentMsgSession
type MsgAgentSession struct {
	*common.AgentSession
	c *MsgAgent
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

// NewProtobufMsgAgent create protobuf message agent client
func NewProtobufMsgAgent(idMsgMapper *IdMsgMapper, options ...options.Option) *MsgAgent {
	return NewMsgAgent(&codec.ProtobufCodec{}, idMsgMapper, options...)
}

// NewJsonMsgAgent create json message agent client
func NewJsonMsgAgent(idMsgMapper *IdMsgMapper, options ...options.Option) *MsgAgent {
	return NewMsgAgent(&codec.JsonCodec{}, idMsgMapper, options...)
}

// NewGobMsgAgent create gob message agent client
func NewGobMsgAgent(idMsgMapper *IdMsgMapper, options ...options.Option) *MsgAgent {
	return NewMsgAgent(&codec.GobCodec{}, idMsgMapper, options...)
}

// NewThriftMsgAgent create thrift message agent client
func NewThriftMsgAgent(idMsgMapper *IdMsgMapper, options ...options.Option) *MsgAgent {
	return NewMsgAgent(&codec.ThriftCodec{}, idMsgMapper, options...)
}

// NewMsgpackMsgAgent create msgpack message agent client
func NewMsgpackMsgAgent(idMsgMapper *IdMsgMapper, options ...options.Option) *MsgAgent {
	return NewMsgAgent(&codec.MsgpackCodec{}, idMsgMapper, options...)
}

// struct AgentMsgServer
type MsgAgentServer struct {
	server  *server.AgentServer
	options MsgServerOptions
}

// NewMsgAgentServer create new message agent server
func NewMsgAgentServer(newFunc NewMsgSessionHandlerFunc, funcArgs []any, codec IMsgCodec, mapper *IdMsgMapper, options ...options.Option) *MsgAgentServer {
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
func (s *MsgAgentServer) Serve() {
	s.server.Serve()
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
func NewProtobufMsgAgentServer(newFunc NewMsgSessionHandlerFunc, funcArgs []any, idMsgMapper *IdMsgMapper, options ...options.Option) *MsgAgentServer {
	return NewMsgAgentServer(newFunc, funcArgs, &codec.ProtobufCodec{}, idMsgMapper, options...)
}

// NewJsonMsgAgentServer create a json message agent server
func NewJsonMsgAgentServerr(newFunc NewMsgSessionHandlerFunc, funcArgs []any, idMsgMapper *IdMsgMapper, options ...options.Option) *MsgAgentServer {
	return NewMsgAgentServer(newFunc, funcArgs, &codec.JsonCodec{}, idMsgMapper, options...)
}

// NewGobMsgAgentServer create a gob message agent server
func NewGobMsgAgentServer(newFunc NewMsgSessionHandlerFunc, funcArgs []any, idMsgMapper *IdMsgMapper, options ...options.Option) *MsgAgentServer {
	return NewMsgAgentServer(newFunc, funcArgs, &codec.GobCodec{}, idMsgMapper, options...)
}

// NewThriftMsgAgentServer create a thrift message agent server
func NewThriftMsgAgentServer(newFunc NewMsgSessionHandlerFunc, funcArgs []any, idMsgMapper *IdMsgMapper, options ...options.Option) *MsgAgentServer {
	return NewMsgAgentServer(newFunc, funcArgs, &codec.ThriftCodec{}, idMsgMapper, options...)
}

// NewMsgpackMsgAgentServer create a msgpack message agent server
func NewMsgpackMsgAgentServer(newFunc NewMsgSessionHandlerFunc, funcArgs []any, idMsgMapper *IdMsgMapper, options ...options.Option) *MsgAgentServer {
	return NewMsgAgentServer(newFunc, funcArgs, &codec.MsgpackCodec{}, idMsgMapper, options...)
}
