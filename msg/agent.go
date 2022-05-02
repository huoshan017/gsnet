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
type AgentMsgClient struct {
	c       *client.AgentClient
	options MsgClientOptions
	codec   IMsgCodec
	mapper  *IdMsgMapper
}

// NewAgentMsgClient  create new agent message client
func NewAgentMsgClient(codec IMsgCodec, mapper *IdMsgMapper, options ...common.Option) *AgentMsgClient {
	return &AgentMsgClient{
		c:      client.NewAgentClient(options...),
		codec:  codec,
		mapper: mapper,
	}
}

// AgentMsgClient.SetConnectHandle  set connect handle
func (c *AgentMsgClient) SetConnectHandle(handle func(*MsgSession)) {
	var h func(common.ISession) = func(sess common.ISession) {
		ms := MsgSession{sess, c.codec, c.mapper, &c.options.MsgOptions}
		handle(&ms)
	}
	c.c.SetConnectHandle(h)
}

// AgentMsgClient.SetDisconnectHandle  set disconnect handle
func (c *AgentMsgClient) SetDisconnectHandle(handle func(*MsgSession, error)) {
	var h func(common.ISession, error) = func(sess common.ISession, err error) {
		ms := MsgSession{sess, c.codec, c.mapper, &c.options.MsgOptions}
		handle(&ms, err)
	}
	c.c.SetDisconnectHandle(h)
}

// AgentMsgClient.SetTickHandle  set tick handle
func (c *AgentMsgClient) SetTickHandle(handle func(*MsgSession, time.Duration)) {
	var h func(common.ISession, time.Duration) = func(sess common.ISession, tick time.Duration) {
		ms := MsgSession{sess, c.codec, c.mapper, &c.options.MsgOptions}
		handle(&ms, tick)
	}
	c.c.SetTickHandle(h)
}

// AgentMsgClient.SetErrorHandle  set error handle
func (c *AgentMsgClient) SetErrorHandle(handle func(error)) {
	c.c.SetErrorHandle(handle)
}

// AgentMsgClient.BoundHandleAndGetAgentSession  bound message handle and create message agent session
func (c *AgentMsgClient) BoundHandleAndGetAgentSession(sess *MsgSession, handle func(*MsgSession, MsgIdType, any) error) *AgentMsgSession {
	var h func(common.ISession, packet.IPacket) error = func(sess common.ISession, pak packet.IPacket) error {
		var ms = MsgSession{sess: sess, codec: c.codec, mapper: c.mapper, options: &c.options.MsgOptions}
		msgid, msgobj, err := ms.splitIdAndMsg(pak.Data())
		if err != nil {
			return err
		}
		return handle(&ms, msgid, msgobj)
	}
	agentSess := c.c.BoundHandleAndGetAgentSession(sess.GetSess(), h)
	return &AgentMsgSession{AgentSession: agentSess, c: c}
}

// struct AgentMsgSession
type AgentMsgSession struct {
	*common.AgentSession
	c *AgentMsgClient
}

// AgentMsgSession.Send
func (c *AgentMsgSession) Send(msgId MsgIdType, msgObj any) error {
	var ms = MsgSession{codec: c.c.codec, mapper: c.c.mapper, options: &c.c.options.MsgOptions}
	dataArray, err := ms.serializeMsg(msgId, msgObj)
	if err != nil {
		return err
	}
	return c.AgentSession.SendBytesArray(dataArray, false)
}

// AgentMsgSession.SendOnCopy
func (c *AgentMsgSession) SendOnCopy(msgId MsgIdType, msgObj any) error {
	var ms = MsgSession{codec: c.c.codec, mapper: c.c.mapper, options: &c.c.options.MsgOptions}
	pData, err := ms.serializeMsgOnCopy(msgId, msgObj)
	if err != nil {
		return err
	}
	return c.AgentSession.SendPoolBuffer(pData)
}

// NewProtoBufAgentMsgClient create protobuf agent message client
func NewProtoBufAgentMsgClient(idMsgMapper *IdMsgMapper, options ...common.Option) *AgentMsgClient {
	return NewAgentMsgClient(&codec.ProtobufCodec{}, idMsgMapper, options...)
}

// NewJsonMsgClient create json agent message client
func NewJsonAgentMsgClient(idMsgMapper *IdMsgMapper, options ...common.Option) *AgentMsgClient {
	return NewAgentMsgClient(&codec.JsonCodec{}, idMsgMapper, options...)
}

// NewGobMsgClient create gob agent message client
func NewGobAgentMsgClient(idMsgMapper *IdMsgMapper, options ...common.Option) *AgentMsgClient {
	return NewAgentMsgClient(&codec.GobCodec{}, idMsgMapper, options...)
}

// NewThriftMsgClient create thrift agent message client
func NewThriftAgentMsgClient(idMsgMapper *IdMsgMapper, options ...common.Option) *AgentMsgClient {
	return NewAgentMsgClient(&codec.ThriftCodec{}, idMsgMapper, options...)
}

// NewMsgpackMsgClient create msgpack agent message client
func NewMsgpackAgentMsgClient(idMsgMapper *IdMsgMapper, options ...common.Option) *AgentMsgClient {
	return NewAgentMsgClient(&codec.MsgpackCodec{}, idMsgMapper, options...)
}

// struct AgentMsgServer
type AgentMsgServer struct {
	server  *server.AgentServer
	options MsgServerOptions
}

// NewMsgServer create new message server
func NewAgentMsgServer(newFunc NewMsgSessionHandlerFunc, funcArgs []any, codec IMsgCodec, mapper *IdMsgMapper, options ...common.Option) *AgentMsgServer {
	s := &AgentMsgServer{}
	newSessionHandlerFunc := prepareMsgServer(newFunc, funcArgs, codec, mapper, &s.options, options...)
	s.server = server.NewAgentServerWithOptions(newSessionHandlerFunc, &s.options.ServerOptions)
	return s
}

// NewProtoBufAgentMsgServer create a protobuf agent message server
func NewProtoBufAgentMsgServer(newFunc NewMsgSessionHandlerFunc, funcArgs []any, idMsgMapper *IdMsgMapper, options ...common.Option) *AgentMsgServer {
	return NewAgentMsgServer(newFunc, funcArgs, &codec.ProtobufCodec{}, idMsgMapper, options...)
}

// NewJsonAgentMsgServer create a json agent message server
func NewJsonAgentMsgServerr(newFunc NewMsgSessionHandlerFunc, funcArgs []any, idMsgMapper *IdMsgMapper, options ...common.Option) *AgentMsgServer {
	return NewAgentMsgServer(newFunc, funcArgs, &codec.JsonCodec{}, idMsgMapper, options...)
}

// NewGobAgentMsgServer create a gob agent message server
func NewGobAgentMsgServer(newFunc NewMsgSessionHandlerFunc, funcArgs []any, idMsgMapper *IdMsgMapper, options ...common.Option) *AgentMsgServer {
	return NewAgentMsgServer(newFunc, funcArgs, &codec.GobCodec{}, idMsgMapper, options...)
}

// NewThriftAgentMsgServer create a thrift agent message server
func NewThriftAgentMsgServer(newFunc NewMsgSessionHandlerFunc, funcArgs []any, idMsgMapper *IdMsgMapper, options ...common.Option) *AgentMsgServer {
	return NewAgentMsgServer(newFunc, funcArgs, &codec.ThriftCodec{}, idMsgMapper, options...)
}

// NewMsgpackAgentMsgServer create a msgpack agent message server
func NewMsgpackAgentMsgServer(newFunc NewMsgSessionHandlerFunc, funcArgs []any, idMsgMapper *IdMsgMapper, options ...common.Option) *AgentMsgServer {
	return NewAgentMsgServer(newFunc, funcArgs, &codec.MsgpackCodec{}, idMsgMapper, options...)
}
