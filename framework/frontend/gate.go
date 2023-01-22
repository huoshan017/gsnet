package frontend

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/huoshan017/gsnet/client"
	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/log"
	"github.com/huoshan017/gsnet/options"
	"github.com/huoshan017/gsnet/packet"
	"github.com/huoshan017/gsnet/server"
)

type RouteType int32

const (
	RouteTypeRandom      RouteType = iota // 随机路由后端服务器一般是无状态的
	RouteTypeFirstRandom RouteType = 1
	RouteTypeSelect      RouteType = 2
)

const (
	DefaultRingLength             int32 = 1024
	DefaultTimeoutDataFromBackend       = 10 * time.Second // 默认从后端返回数据的超时
	DefaultCheckCacheDataTick           = time.Second      // 默认tick间隔
)

type ring struct {
	list        []int32 // agent id
	rn, wn, cnt int32
}

func newRing() *ring {
	return newRingWithLength(DefaultRingLength)
}

func newRingWithLength(length int32) *ring {
	return &ring{
		list: make([]int32, length),
		rn:   -1,
		wn:   -1,
	}
}

func (r *ring) write(id int32) bool {
	if int(r.cnt) >= len(r.list) {
		return false
	}
	if r.wn < 0 {
		r.wn += 1
	} else {
		r.wn = (r.wn + 1) % int32(len(r.list))
	}
	r.list[r.wn] = id
	r.cnt += 1
	return true
}

func (r ring) peek() (int32, bool) {
	if r.cnt <= 0 {
		return -1, false
	}
	var (
		id int32
	)
	if int(r.rn)+1 >= len(r.list) {
		id = r.list[0]
	} else {
		id = r.list[r.rn+1]
	}
	return id, true
}

func (r *ring) advance() {
	r.rn += 1
	if int(r.rn) >= len(r.list) {
		r.rn = 0
	}
	r.cnt -= 1
}

type dnode struct {
	data []byte
	next *dnode
}

func get_dnode() *dnode {
	return dnodePool.Get().(*dnode)
}

func put_dnode(n *dnode) {
	n.data = nil
	n.next = nil
	dnodePool.Put(n)
}

type dlist struct {
	head, tail *dnode
	length     int32
}

func newDlist() *dlist {
	return &dlist{}
}

func (l *dlist) pushBack(data []byte) {
	node := get_dnode() //&dnode{data: data}
	node.data = data
	if l.head == nil {
		l.head = node
		l.tail = l.head
	} else {
		l.tail.next = node
		l.tail = node
	}
	l.length += 1
}

func (l *dlist) popFront() ([]byte, bool) {
	d, o := l.peek()
	if !o {
		return nil, false
	}
	l.removeFront()
	return d, true
}

func (l *dlist) peek() ([]byte, bool) {
	if l.head == nil {
		return nil, false
	}
	return l.head.data, true
}

func (l *dlist) removeFront() {
	n := l.head
	if l.tail == l.head {
		l.head = nil
		l.tail = nil
	} else {
		l.head = l.head.next
	}
	l.length -= 1
	put_dnode(n)
}

func (l *dlist) getLength() int32 {
	return l.length
}

type agentDataListMap map[int32]*dlist

func newAgentDataListMap() agentDataListMap {
	return agentDataListMap(make(map[int32]*dlist))
}

func (sm agentDataListMap) pushBack(id int32, data []byte) {
	list, o := sm[id]
	if !o {
		list = newDlist()
		(sm)[id] = list
	}
	list.pushBack(data)
}

func (sm agentDataListMap) popFront(id int32) ([]byte, bool) {
	list, o := sm[id]
	if !o {
		return nil, false
	}
	return list.popFront()
}

func (sm agentDataListMap) length() int32 {
	var l int32
	for _, m := range sm {
		l += m.getLength()
	}
	return l
}

type BackendInfo struct {
	Id      int32
	Address string
	ConnNum int32
}

type GateOptions struct {
	backendList []BackendInfo
	routeType   RouteType
}

func NewGateOptions(backendList []BackendInfo, routType RouteType) *GateOptions {
	return &GateOptions{backendList: backendList, routeType: routType}
}

type gateSessionHandler struct {
	routeType       RouteType
	agentGroup      *client.AgentGroup
	agentSessionMap map[int32]*common.AgentSession

	// used to first random
	agentId int32

	// used to random
	sequenceIds       *ring
	agentCacheDataMap agentDataListMap
}

func (h *gateSessionHandler) OnConnect(sess common.ISession) {
	log.Infof("session %v connected", sess.GetId())
}

func (h *gateSessionHandler) OnReady(sess common.ISession) {
	h.agentSessionMap = h.agentGroup.BoundSession(sess, h.OnPacketFromBackEnd)
	if h.routeType == RouteTypeRandom {
		h.sequenceIds = newRing()
		h.agentCacheDataMap = newAgentDataListMap()
	}
	log.Infof("session %v ready", sess.GetId())
}

func (h *gateSessionHandler) OnDisconnect(sess common.ISession, err error) {
	h.agentGroup.UnboundSession(sess, h.agentSessionMap)
	log.Infof("session %v disconnected, err %v", sess.GetId(), err)
}

func (h *gateSessionHandler) OnPacket(sess common.ISession, pak packet.IPacket) error {
	var agentId int32
	switch h.routeType {
	case RouteTypeRandom:
		agentId = h.agentGroup.RandomGetId()
	case RouteTypeFirstRandom:
		if h.agentId <= 0 {
			h.agentId = h.agentGroup.RandomGetId()
		}
		agentId = h.agentId
	case RouteTypeSelect:
	}

	agentSess, o := h.agentSessionMap[agentId]
	if !o {
		log.Infof("gate handler cant get AgentSession by id %v", agentId)
		return nil
	}
	agentClient := h.agentGroup.Get(agentId)
	if agentClient == nil {
		log.Infof("cant get agentClient by id %v", agentId)
		return nil
	}

	err := agentSess.Send(pak.Data(), func() bool {
		return pak.MMType() != packet.MemoryManagementSystemGC
	}())

	// 同一Session的逻辑在一个goroutine中执行，所以其"发送消息"的顺序可以保证
	// 当从多个后端随机选择一个向其发送消息时，"返回的消息"顺序是无法确定的
	// 因此需要保存一个发送的队列，对返回的消息排序后再回给客户端
	// 这个队列只要记录后端服务器对应的代理客户端id，就能保证顺序性
	if err == nil && h.routeType == RouteTypeRandom {
		// 记录发送的代理客户端id
		if !h.sequenceIds.write(agentId) {
			panic("accumulate ids is full")
		}
	}
	return err
}

func (h *gateSessionHandler) OnTick(sess common.ISession, tick time.Duration) {
}

func (h *gateSessionHandler) OnError(err error) {
	log.Infof("session err %v", err)
}

func (h *gateSessionHandler) OnPacketFromBackEnd(sess common.ISession, agentId int32, pak packet.IPacket) error {
	var (
		data     []byte
		copyData bool
	)
	if h.routeType == RouteTypeRandom {
		var (
			aid int32
			o   bool
		)
		aid, o = h.sequenceIds.peek()
		if !o {
			panic("gate: cant get sequence id for packet from backend")
		}
		// 先缓存起来
		if aid != agentId {
			if pak.MMType() == packet.MemoryManagementSystemGC {
				h.agentCacheDataMap.pushBack(agentId, pak.Data())
			} else {
				d := make([]byte, len(pak.Data()))
				copy(d, pak.Data())
				h.agentCacheDataMap.pushBack(agentId, d)
			}
			return nil
		}
		data, o = h.agentCacheDataMap.popFront(aid)
		if !o {
			data = pak.Data()
			copyData = pak.MMType() != packet.MemoryManagementSystemGC
		}
		h.sequenceIds.advance()
	} else {
		data = pak.Data()
		copyData = pak.MMType() != packet.MemoryManagementSystemGC
	}
	err := sess.Send(data, copyData)
	if err == nil && h.routeType == RouteTypeRandom {
		err = h.checkSendCacheData(sess)
	}
	return err
}

func (h *gateSessionHandler) checkSendCacheData(sess common.ISession) error {
	var err error
	if h.agentCacheDataMap.length() > 0 {
		var (
			aid  int32
			data []byte
			o    bool
		)
		for err == nil {
			aid, o = h.sequenceIds.peek()
			if !o {
				break
			}
			data, o = h.agentCacheDataMap.popFront(aid)
			if !o {
				break
			}
			h.sequenceIds.advance()
			err = sess.Send(data, false)
		}
	}
	return err
}

func newGateSessionHandler(args ...any) common.ISessionHandler {
	agentGroup := args[0].(*client.AgentGroup)
	routeType := args[1].(RouteType)
	return &gateSessionHandler{
		routeType:  routeType,
		agentGroup: agentGroup,
	}
}

type Gate struct {
	serv       *server.Server
	options    *GateOptions
	agentGroup *client.AgentGroup
}

func NewGate(goptions *GateOptions, ops ...options.Option) *Gate {
	var gate = &Gate{
		options:    goptions,
		agentGroup: client.NewAgentGroup(getNextAgentGroupId()),
	}
	gate.serv = server.NewServer(newGateSessionHandler, options.WithNewSessionHandlerFuncArgs(gate.agentGroup, goptions.routeType), options.WithAutoReconnect(true), options.WithTickSpan(DefaultCheckCacheDataTick))
	return gate
}

func (g *Gate) ListenAndServe(address string) error {
	var addressList []string
	for i := 0; i < len(g.options.backendList); i++ {
		bi := g.options.backendList[i]
		for j := 0; j < int(bi.ConnNum); j++ {
			addressList = append(addressList, bi.Address)
		}
	}
	g.agentGroup.DialAsync(addressList, 0, func(err error) {
		log.Infof("gate: dial err: %v", err)
	})
	return g.serv.ListenAndServe(address)
}

var (
	agentIdCounter int32
	dnodePool      *sync.Pool
)

func getNextAgentGroupId() int32 {
	return atomic.AddInt32(&agentIdCounter, 1)
}

func init() {
	dnodePool = &sync.Pool{
		New: func() any {
			return &dnode{}
		},
	}
}
