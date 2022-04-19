package packet

import (
	"sync"

	"github.com/huoshan017/gsnet/pool"
)

// pool for type `Packet`
type DefaultPacketPool struct {
	pool         *sync.Pool
	usingPackets sync.Map // 保存分配的Packet地址
}

func NewDefaultPacketPool() *DefaultPacketPool {
	return &DefaultPacketPool{
		pool: &sync.Pool{
			New: func() any {
				return &Packet{}
			},
		},
	}
}

func (p *DefaultPacketPool) Get() IPacket {
	pak := p.pool.Get().(IPacket)
	p.usingPackets.Store(pak, true)
	return pak
}

func (p *DefaultPacketPool) Put(pak IPacket) {
	// 不是*Packet类型
	if _, o := any(pak).(*Packet); !o {
		return
	}
	if _, o := p.usingPackets.LoadAndDelete(pak); !o {
		return
	}
	if pak.MMType() == MemoryManagementPoolFrameworkFree || pak.MMType() == MemoryManagementPoolUserManualFree {
		pool.GetBuffPool().Free(pak.PData())
	}
	p.pool.Put(pak)
}

// 默认的packet池和packet构建器
var (
	defaultPacketPool *DefaultPacketPool
)

func init() {
	defaultPacketPool = NewDefaultPacketPool()
}

func GetDefaultPacketPool() *DefaultPacketPool {
	return defaultPacketPool
}
