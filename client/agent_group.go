package client

import (
	"math/rand"
	"sync"
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/log"
	"github.com/huoshan017/gsnet/options"
	"github.com/huoshan017/gsnet/packet"
)

const (
	DefaultAgentGroupSize = 32
)

type AgentGroup struct {
	id          int32
	agentIds    []int32
	agentLength int32
	agentsMap   map[int32]*struct {
		agent *Agent
		index int32
	}
	locker sync.RWMutex
	rand   *rand.Rand
}

func NewAgentGroupWithCount(id int32, maxCount int32) *AgentGroup {
	return &AgentGroup{
		id:       id,
		agentIds: make([]int32, maxCount),
		rand:     rand.New(rand.NewSource(time.Now().UnixNano() + int64(id))),
		agentsMap: make(map[int32]*struct {
			agent *Agent
			index int32
		}),
	}
}

func NewAgentGroup(id int32) *AgentGroup {
	return NewAgentGroupWithCount(id, DefaultAgentGroupSize)
}

func (ag *AgentGroup) DialAsync(addressList []string, timeout time.Duration, callback func(error)) {
	for i := 0; i < len(addressList); i++ {
		agent := NewAgent(options.WithAutoReconnect(true))
		agent.SetConnectHandle(func(sess common.ISession) {
			connGetter, o := sess.(common.IConnGetter)
			if o {
				log.Infof("agent session(%v) connected %v", sess.GetId(), connGetter.Conn().RemoteAddr())
			} else {
				log.Infof("agent session(%v) connected", sess.GetId())
			}
		})
		agent.SetReadyHandle(func(sess common.ISession) {
			ag.addAgent(agent)
			log.Infof("agent session(%v) ready", sess.GetId())
		})
		agent.SetDisconnectHandle(func(sess common.ISession, err error) {
			ag.removeAgent(agent.id)
			log.Infof("agent session(%v) disconnected, err %v", sess.GetId(), err)
		})
		agent.SetTickHandle(func(sess common.ISession, tick time.Duration) {
		})
		agent.SetErrorHandle(func(err error) {
			log.Infof("agent err %v", err)
		})
		agent.DialAsync(addressList[i], timeout, callback)
	}
}

func (ag *AgentGroup) Get(id int32) *Agent {
	ag.locker.RLock()
	defer ag.locker.RUnlock()
	return ag.agentsMap[id].agent
}

func (ag *AgentGroup) RandomGet() *Agent {
	ag.locker.RLock()
	defer ag.locker.RUnlock()
	if ag.agentLength <= 0 {
		return nil
	}
	rn := ag.rand.Int31n(ag.agentLength)
	return ag.agentsMap[ag.agentIds[rn]].agent
}

func (ag *AgentGroup) RandomGetId() int32 {
	ag.locker.RLock()
	defer ag.locker.RUnlock()
	rn := ag.rand.Int31n(ag.agentLength)
	return ag.agentIds[rn]
}

func (ag *AgentGroup) BoundSession(sess common.ISession, handle func(common.ISession, int32, packet.IPacket) error) map[int32]*common.AgentSession {
	ag.locker.RLock()
	defer ag.locker.RUnlock()

	var sessMap = make(map[int32]*common.AgentSession)
	for _, ac := range ag.agentsMap {
		agentSess := ac.agent.BoundServerSession(sess, handle)
		sessMap[ac.agent.id] = agentSess
	}
	return sessMap
}

func (ag *AgentGroup) UnboundSession(sess common.ISession, sessMap map[int32]*common.AgentSession) {
	ag.locker.RLock()
	defer ag.locker.RUnlock()

	for aid, as := range sessMap {
		agentIndex, o := ag.agentsMap[aid]
		if !o {
			log.Infof("agent group cant get agent by id %v", aid)
			continue
		}
		agentIndex.agent.UnboundServerSession(sess, as)
	}
}

func (ag *AgentGroup) addAgent(agent *Agent) bool {
	ag.locker.Lock()
	defer ag.locker.Unlock()
	if int(ag.agentLength) >= len(ag.agentIds) {
		return false
	}
	if _, o := ag.agentsMap[agent.id]; o {
		return false
	}
	ag.agentIds[ag.agentLength] = agent.id
	ag.agentsMap[agent.id] = &struct {
		agent *Agent
		index int32
	}{agent, ag.agentLength}
	ag.agentLength += 1
	return true
}

func (ag *AgentGroup) removeAgent(id int32) bool {
	ag.locker.Lock()
	defer ag.locker.Unlock()
	var (
		v *struct {
			agent *Agent
			index int32
		}
		o bool
	)
	if v, o = ag.agentsMap[id]; !o {
		return false
	}
	delete(ag.agentsMap, id)
	if v.index != ag.agentLength-1 {
		moveId := ag.agentIds[ag.agentLength-1]
		ag.agentIds[v.index] = moveId
		vIndex := v.index
		if v, o = ag.agentsMap[moveId]; o {
			v.index = vIndex
		}
	}
	ag.agentLength -= 1
	return true
}
