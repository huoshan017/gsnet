package packet

import (
	"sync"

	"github.com/huoshan017/gsnet/pool"
)

type PoolPacketType uint8

const (
	PoolPacketNormal PoolPacketType = iota
	PoolPacketShared PoolPacketType = 1
)

// pool for type `Packet`
type DefaultPacketPool struct {
	pool         *sync.Pool
	sharedPool   *sync.Pool
	usingPackets sync.Map // 保存分配的Packet地址
}

func NewDefaultPacketPool() *DefaultPacketPool {
	return &DefaultPacketPool{
		pool: &sync.Pool{
			New: func() any {
				return &Packet{}
			},
		},
		sharedPool: &sync.Pool{
			New: func() any {
				return &SharedPacket{}
			},
		},
	}
}

func (p *DefaultPacketPool) Get() IPacket {
	pak := p.pool.Get().(*Packet)
	pak.Reset()
	p.usingPackets.Store(pak, true)
	return pak
}

func (p *DefaultPacketPool) GetWithType(typ PoolPacketType) IPacket {
	if typ == PoolPacketNormal {
		return p.Get()
	} else if typ == PoolPacketShared {
		var pak = p.sharedPool.Get().(*SharedPacket)
		p.usingPackets.Store(pak, true)
		return pak
	}
	return nil
}

func (p *DefaultPacketPool) Put(pak IPacket) {
	pak.Release()
	if _, o := p.usingPackets.LoadAndDelete(pak); !o {
		return
	}
	// 不是*Packet类型
	if pt, o := pak.(*Packet); o {
		if pt.MMType() == MemoryManagementPoolFrameworkFree || pt.MMType() == MemoryManagementPoolUserManualFree {
			if pt.PData() != nil {
				pool.GetBuffPool().Free(pt.PData())
			}
		}
		p.pool.Put(pak)
	} else if sp, o := pak.(*SharedPacket); o {
		p.sharedPool.Put(sp)
	}
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
