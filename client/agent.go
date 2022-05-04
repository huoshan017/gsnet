package client

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/log"
	"github.com/huoshan017/gsnet/packet"
	"github.com/huoshan017/gsnet/pool"
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

func (h *commonHandler) Send(sess common.ISession, id uint64, data []byte) error {
	temp := make([]byte, 8)
	common.Uint64ToBuffer(id, temp)
	return sess.SendBytesArray([][]byte{temp, data}, false)
}

func (h *commonHandler) SendOnCopy(sess common.ISession, id uint64, data []byte) error {
	buffer := pool.GetBuffPool().Alloc(8 + int32(len(data)))
	common.Uint64ToBuffer(id, (*buffer)[:8])
	copy((*buffer)[8:], data)
	return sess.SendPoolBuffer(buffer)
}

type clientHandler struct {
	commonHandler
	owner    *AgentClient
	idPacket common.IdWithPacket
}

func newClientHandler(c *AgentClient) *clientHandler {
	return &clientHandler{owner: c}
}

func (h *clientHandler) OnPacket(sess common.ISession, pak packet.IPacket) error {
	agentId := common.BufferToUint32(pak.Data()[:4])
	chPak := h.owner.getPakChan(agentId)
	if chPak == nil {
		log.Infof("gsnet: not yet bound handle for agent %v", agentId)
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

type AgentClient struct {
	c        *Client
	handler  *clientHandler
	id       int32
	pakChans sync.Map
}

func NewAgentClient(options ...common.Option) *AgentClient {
	c := &AgentClient{id: getNextAgentClientId()}
	c.handler = newClientHandler(c)
	c.c = NewClient(c.handler, options...)
	return c
}

func (c *AgentClient) Dial(address string) error {
	if err := c.c.Connect(address); err != nil {
		return err
	}
	go c.c.Run()
	return nil
}

func (c *AgentClient) DialTimeout(address string, timeout time.Duration) error {
	if err := c.c.ConnectWithTimeout(address, timeout); err != nil {
		return err
	}
	go c.c.Run()
	return nil
}

func (c *AgentClient) BoundSession(sess common.ISession, handle func(common.ISession, packet.IPacket) error) *common.AgentSession {
	sess.AddInboundHandle(c.id, handle)
	agentSessionId := getNextAgentSessionId()
	c.pakChans.Store(agentSessionId, sess.GetPacketChannel())
	return common.NewAgentSession(agentSessionId, c.c.GetSession())
}

func (c *AgentClient) UnboundSession(sess common.ISession, asess *common.AgentSession) {
	sess.RemoveInboundHandle(c.id)
	v, o := c.pakChans.Load(asess.GetAgentId())
	if !o {
		log.Infof("gsnet: cant get packet channel with id %v for agent client", asess.GetAgentId())
		return
	}
	if v != sess.GetPacketChannel() {
		log.Infof("gsnet: to delete packet channel with id %v dismatch to specified", asess.GetAgentId())
		return
	}
	c.pakChans.Delete(asess.GetAgentId())
}

func (c *AgentClient) SetConnectHandle(handle func(common.ISession)) {
	c.handler.setConnectHandle(handle)
}

func (c *AgentClient) SetDisconnectHandle(handle func(common.ISession, error)) {
	c.handler.setDisconnectHandle(handle)
}

func (c *AgentClient) SetTickHandle(handle func(common.ISession, time.Duration)) {
	c.handler.setTickHandle(handle)
}

func (c *AgentClient) SetErrorHandle(handle func(error)) {
	c.handler.setErrorHandle(handle)
}

func (c *AgentClient) getPakChan(agentId uint32) chan common.IdWithPacket {
	var (
		d any
		o bool
	)
	if d, o = c.pakChans.Load(agentId); !o {
		return nil
	}
	return d.(chan common.IdWithPacket)
}

type AgentClientManager struct {
	clients map[int32]*AgentClient
	name2Id map[string]int32
	locker  sync.RWMutex
}

func NewAgentClientManager() *AgentClientManager {
	return &AgentClientManager{
		clients: make(map[int32]*AgentClient),
		name2Id: make(map[string]int32),
	}
}

func (m *AgentClientManager) NewClient(name string, options ...common.Option) *AgentClient {
	client := NewAgentClient(options...)
	m.addClient(name, client)
	return client
}

func (m *AgentClientManager) addClient(name string, c *AgentClient) {
	m.locker.Lock()
	m.clients[c.id] = c
	m.name2Id[name] = c.id
	m.locker.Unlock()
}

func (m *AgentClientManager) GetClient(name string) *AgentClient {
	m.locker.RLock()
	defer m.locker.RUnlock()
	id := m.name2Id[name]
	return m.clients[id]
}

var (
	globalAgentClientIdCounter  int32
	globalAgentSessionIdCounter uint32
)

func getNextAgentClientId() int32 {
	return atomic.AddInt32(&globalAgentClientIdCounter, 1)
}

func getNextAgentSessionId() uint32 {
	return atomic.AddUint32(&globalAgentSessionIdCounter, 1)
}
