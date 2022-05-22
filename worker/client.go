package worker

import (
	"sync"
	"time"

	"github.com/huoshan017/gsnet/client"
	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/log"
	"github.com/huoshan017/gsnet/packet"
)

type clientHandler struct {
	commonHandler
	owner    *Client
	idPacket common.IdWithPacket
}

func newClientHandler(c *Client) *clientHandler {
	return &clientHandler{owner: c}
}

func (h *clientHandler) OnPacket(sess common.ISession, pak packet.IPacket) error {
	sessId := common.BufferToUint64(pak.Data()[:8])
	chPak := h.owner.getPakChan(sessId)
	if chPak == nil {
		log.Infof("gsnet: not yet bound handle for session %v", sessId)
		return nil
	}
	if pak.MMType() == packet.MemoryManagementSystemGC {
		h.idPacket.Set(h.owner.id, pak)
	} else {
		ppak, o := pak.(*packet.Packet)
		if o {
			newPak := packet.GetDefaultPacketPool().Get()
			ppak.ChangeDataOwnership(newPak.(*packet.Packet), 8, packet.MemoryManagementPoolUserManualFree)
			h.idPacket.Set(h.owner.id, newPak)
		} else {
			if _, o = pak.(*packet.BytesPacket); o {
				h.idPacket.Set(h.owner.id, pak)
			} else {
				return ErrPacketTypeNotSupported
			}
		}
	}
	chPak <- h.idPacket
	return nil
}

type Client struct {
	c        *client.Client
	handler  *clientHandler
	id       int32
	pakChans sync.Map
}

func newClient(options ...common.Option) *Client {
	c := &Client{}
	c.handler = newClientHandler(c)
	c.c = client.NewClient(c.handler, options...)
	return c
}

func (c *Client) Dial(address string) error {
	if err := c.c.Connect(address); err != nil {
		return err
	}
	go c.c.Run()
	return nil
}

func (c *Client) DialTimeout(address string, timeout time.Duration) error {
	if err := c.c.ConnectWithTimeout(address, timeout); err != nil {
		return err
	}
	go c.c.Run()
	return nil
}

func (c *Client) BoundPacketHandle(sess common.ISession, handle func(common.ISession, int32, packet.IPacket) error) {
	sess.AddInboundHandle(c.id, handle)
	c.pakChans.Store(sess.GetId(), sess.GetPacketChannel())
}

func (c *Client) NewSessionChannel(sess common.ISession) *common.SessionChannel {
	return common.NewSessionChannel(sess.GetId(), c.c.GetSession())
}

func (c *Client) SetConnectHandle(handle func(common.ISession)) {
	c.handler.setConnectHandle(handle)
}

func (c *Client) SetReadyHandle(handle func(common.ISession)) {
	c.handler.setReadyHandle(handle)
}

func (c *Client) SetDisconnectHandle(handle func(common.ISession, error)) {
	c.handler.setDisconnectHandle(handle)
}

func (c *Client) SetTickHandle(handle func(common.ISession, time.Duration)) {
	c.handler.setTickHandle(handle)
}

func (c *Client) SetErrorHandle(handle func(error)) {
	c.handler.setErrorHandle(handle)
}

func (c *Client) getPakChan(id uint64) chan common.IdWithPacket {
	var (
		d any
		o bool
	)
	if d, o = c.pakChans.Load(id); !o {
		return nil
	}
	return d.(chan common.IdWithPacket)
}

type clientManager struct {
	clients   map[int32]*Client
	name2Id   map[string]int32
	idCounter int32
	locker    sync.RWMutex
}

func newClientManager() *clientManager {
	return &clientManager{
		clients: make(map[int32]*Client),
		name2Id: make(map[string]int32),
	}
}

func (cm *clientManager) addClient(name string, c *Client) {
	cm.locker.Lock()
	cm.idCounter += 1
	cm.clients[cm.idCounter] = c
	c.id = cm.idCounter
	cm.name2Id[name] = cm.idCounter
	cm.locker.Unlock()
}

func (cm *clientManager) getClientById(id int32) *Client {
	cm.locker.RLock()
	defer cm.locker.RUnlock()
	return cm.clients[id]
}

func (cm *clientManager) getClient(name string) *Client {
	cm.locker.RLock()
	defer cm.locker.RUnlock()
	id := cm.name2Id[name]
	return cm.clients[id]
}

func NewClient(name string, options ...common.Option) *Client {
	c := newClient(options...)
	cmInstance.addClient(name, c)
	return c
}

func NewClientAndConnect(name string, address string, options ...common.Option) (*Client, error) {
	var (
		c   *Client
		err error
	)
	c = NewClient(name, options...)
	if err = c.c.Connect(address); err != nil {
		return nil, err
	}
	cmInstance.addClient(name, c)
	go c.c.Run()
	return c, nil
}

func GetClient(name string) *Client {
	return cmInstance.getClient(name)
}

func GetClientById(id int32) *Client {
	return cmInstance.getClientById(id)
}

var cmInstance *clientManager

func init() {
	cmInstance = newClientManager()
}
