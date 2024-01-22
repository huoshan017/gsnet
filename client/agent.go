package client

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/log"
	"github.com/huoshan017/gsnet/options"
	"github.com/huoshan017/gsnet/packet"
)

type commonHandler struct {
	connectHandle    func(common.ISession)
	readyHandle      func(common.ISession)
	disconnectHandle func(common.ISession, error)
	tickHandle       func(common.ISession, time.Duration)
	errorHandle      func(error)
}

func (h *commonHandler) setConnectHandle(handle func(common.ISession)) {
	h.connectHandle = handle
}

func (h *commonHandler) setReadyHandle(handle func(common.ISession)) {
	h.readyHandle = handle
}

func (h *commonHandler) setDisconnectHandle(handle func(common.ISession, error)) {
	h.disconnectHandle = handle
}

func (h *commonHandler) setTickHandle(handle func(common.ISession, time.Duration)) {
	h.tickHandle = handle
}

func (h *commonHandler) setErrorHandle(handle func(error)) {
	h.errorHandle = handle
}

func (h *commonHandler) OnConnect(sess common.ISession) {
	if h.connectHandle != nil {
		h.connectHandle(sess)
	}
}

func (h *commonHandler) OnReady(sess common.ISession) {
	if h.readyHandle != nil {
		h.readyHandle(sess)
	}
}

func (h *commonHandler) OnTick(sess common.ISession, tick time.Duration) {
	if h.tickHandle != nil {
		h.tickHandle(sess, tick)
	}
}

func (h *commonHandler) OnDisconnect(sess common.ISession, err error) {
	if h.disconnectHandle != nil {
		h.disconnectHandle(sess, err)
	}
}

func (h *commonHandler) OnError(err error) {
	if h.errorHandle != nil {
		h.errorHandle(err)
	}
}

type clientHandler struct {
	commonHandler
	owner    *Agent
	idPacket common.IdWithPacket
}

func newClientHandler(c *Agent) *clientHandler {
	return &clientHandler{owner: c}
}

func (h *clientHandler) OnPacket(sess common.ISession, pak packet.IPacket) error {
	agentSessionId := common.BufferToUint32(pak.Data()[:4])
	chPak := h.owner.getPakChan(agentSessionId)
	if chPak == nil {
		log.Infof("gsnet: not yet bound handle for agent %v", agentSessionId)
		return nil
	}
	if pak.MMType() == packet.MemoryManagementSystemGC {
		h.idPacket.Set(h.owner.id, pak)
	} else {
		ppak, o := pak.(*packet.Packet)
		if o {
			newPak := packet.GetDefaultPacketPool().Get()
			ppak.ChangeDataOwnership(newPak.(*packet.Packet), 4, packet.MemoryManagementPoolUserManualFree)
			h.idPacket.Set(h.owner.id, newPak)
		} else {
			if _, o = pak.(*packet.BytesPacket); o {
				h.idPacket.Set(h.owner.id, pak)
			} else {
				return common.ErrPacketTypeNotSupported
			}
		}
	}
	chPak <- h.idPacket
	return nil
}

type Agent struct {
	id       int32
	c        *Client
	handler  *clientHandler
	pakChans sync.Map
}

func NewAgent(options ...options.Option) *Agent {
	c := &Agent{id: getNextAgentId()}
	c.handler = newClientHandler(c)
	c.c = NewClient(c.handler, options...)
	return c
}

func (c *Agent) Dial(address string) error {
	if err := c.c.Connect(address); err != nil {
		return err
	}
	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.WithStack(err)
			}
		}()
		c.c.Run()
	}()
	return nil
}

func (c *Agent) DialTimeout(address string, timeout time.Duration) error {
	if err := c.c.ConnectWithTimeout(address, timeout); err != nil {
		return err
	}
	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.WithStack(err)
			}
		}()
		c.c.Run()
	}()
	return nil
}

func (c *Agent) DialAsync(address string, timeout time.Duration, callback func(error)) {
	c.c.ConnectAsync(address, timeout, callback)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.WithStack(err)
			}
		}()
		c.c.Run()
	}()
}

func (c *Agent) Close() {
	c.c.Close()
}

func (c *Agent) CloseWait(secs int) {
	c.c.CloseWait(secs)
}

func (c *Agent) GetId() int32 {
	return c.id
}

func (c *Agent) IsNotConnect() bool {
	return c.c.IsNotConnect()
}

func (c *Agent) IsConnecting() bool {
	return c.c.IsConnecting()
}

func (c *Agent) IsConnected() bool {
	return c.c.IsConnected()
}

func (c *Agent) IsReady() bool {
	return c.c.IsReady()
}

func (c *Agent) IsDisconnecting() bool {
	return c.c.IsDisconnecting()
}

func (c *Agent) IsDisconnected() bool {
	return c.c.IsDisconnected()
}

func (c *Agent) BoundServerSession(sess common.ISession, handle func(common.ISession, int32, packet.IPacket) error) *common.AgentSession {
	sess.AddInboundHandle(c.id, handle)
	agentSessionId := getNextAgentSessionId()
	c.pakChans.Store(agentSessionId, sess.GetPacketChannel())
	return common.NewAgentSession(agentSessionId, c.c.GetSession())
}

func (c *Agent) UnboundServerSession(sess common.ISession, asess *common.AgentSession) {
	sess.RemoveInboundHandle(c.id)
	c.pakChans.Delete(asess.GetAgentId())
}

func (c *Agent) SetConnectHandle(handle func(common.ISession)) {
	c.handler.setConnectHandle(handle)
}

func (c *Agent) SetReadyHandle(handle func(common.ISession)) {
	c.handler.setReadyHandle(handle)
}

func (c *Agent) SetDisconnectHandle(handle func(common.ISession, error)) {
	c.handler.setDisconnectHandle(handle)
}

func (c *Agent) SetTickHandle(handle func(common.ISession, time.Duration)) {
	c.handler.setTickHandle(handle)
}

func (c *Agent) SetErrorHandle(handle func(error)) {
	c.handler.setErrorHandle(handle)
}

func (c *Agent) getPakChan(agentId uint32) chan common.IdWithPacket {
	var (
		d any
		o bool
	)
	if d, o = c.pakChans.Load(agentId); !o {
		return nil
	}
	return d.(chan common.IdWithPacket)
}

type AgentManager struct {
	clients map[int32]*Agent
	name2Id map[string]int32
	locker  sync.RWMutex
}

func NewAgentManager() *AgentManager {
	return &AgentManager{
		clients: make(map[int32]*Agent),
		name2Id: make(map[string]int32),
	}
}

func (m *AgentManager) NewClient(name string, options ...options.Option) *Agent {
	client := NewAgent(options...)
	m.addClient(name, client)
	return client
}

func (m *AgentManager) addClient(name string, c *Agent) {
	m.locker.Lock()
	m.clients[c.id] = c
	m.name2Id[name] = c.id
	m.locker.Unlock()
}

func (m *AgentManager) GetClient(name string) *Agent {
	m.locker.RLock()
	defer m.locker.RUnlock()
	id := m.name2Id[name]
	return m.clients[id]
}

var (
	globalAgentIdCounter        int32
	globalAgentSessionIdCounter uint32
)

func getNextAgentId() int32 {
	return atomic.AddInt32(&globalAgentIdCounter, 1)
}

func getNextAgentSessionId() uint32 {
	return atomic.AddUint32(&globalAgentSessionIdCounter, 1)
}
