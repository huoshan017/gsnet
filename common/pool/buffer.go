package pool

import "sync"

const (
	// 每层数量
	oneLayerSize = int32(8)
	// 最大缓存大小
	maxBufSize = int32(1 << 17)
)

var (
	// 每层的步进大小
	layerStepSizeArray = []int32{
		1 << 2, 1 << 3, 1 << 4, 1 << 5, 1 << 6, 1 << 7, 1 << 8, 1 << 9, 1 << 10, 1 << 11, 1 << 12, 1 << 13,
	}
	// 每层的开始大小
	layerStartSizeArray = []int32{
		1 << 5, 1 << 6, 1 << 7, 1 << 8, 1 << 9, 1 << 10, 1 << 11, 1 << 12, 1 << 13, 1 << 14, 1 << 15, 1 << 16,
	}
	/*
		bufSizeArray = [][layerSizeCount]int32{
			{ 1<<4, 18, 20, 22, 24, 26, 28, 30 },
			{ 1<<5, 36, 40, 44, 48, 52, 56, 60 },
			{ 1<<6, 72, 80, 88, 96, 104, 112, 120 },
			{ 1<<7, 144, 160, 176, 192, 208, 224, 240, },
			{ 1<<8, 288, 320, 352, 384, 416, 448, 480, },
			{ 1<<9, 512 + 64, 512 + 128, 512 + 192, 512 + 256, 512 + 320, 512 + 384, 512 + 448, },
			{ 1<<10, 1024 + 128, 1024 + 256, 1024 + 384, 1024 + 512, 1024 + 640, 1024 + 768, 1024 + 896, },
			{ 1<<11, 2048 + 256, 2048 + 512, 2048 + 768, 2048 + 1024, 2048 + 1280, 2048 + 1536, 2048 + 256*7, },
			{ 1<<12, 4096 + 512, 4096 + 512*2, 4096 + 512*3, 4096 + 512*4, 4096 + 512*5, 4096 + 512*6, 4096 + 512*7, },
			{ 1<<13, 8192 + 1024, 8192 + 1024*2, 8192 + 1024*3, 8192 + 1024*4, 8192 + 1024*5, 8192 + 1024*6, 8192 + 1024*7, },
			{ 1<<14, 16384 + 2048, 16384 + 2048*2, 16384 + 2048*3, 16384 + 2048*4, 16384 + 2048*5, 16384 + 2048*6, 16384 + 2048*7, },
			{ 1<<15, 32768 + 4096, 32768 + 4096*2, 32768 + 4096*3, 32768 + 4096*4, 32768 + 4096*5, 32768 + 4096*6, 32768 + 4096*7, },
			{ 1<<16, 65536 + 8192, 65536 + 8192*2, 65536 + 8192*3, 65536 + 8192*4, 65536 + 8192*5, 65536 + 8192*6, 65536 + 8192*7, },
			{ 1<<17, },
		}
	*/
)

// BufferPool struct
type BufferPool struct {
	buffSizeArray [][oneLayerSize]int32
	pools         [][oneLayerSize]*sync.Pool
	maxSizePool   *sync.Pool
}

// NewBufferPool create new BufferPool instance
func NewBufferPool() *BufferPool {
	buffPool := &BufferPool{
		buffSizeArray: make([][oneLayerSize]int32, len(layerStartSizeArray)),
		pools:         make([][oneLayerSize]*sync.Pool, len(layerStartSizeArray)),
		maxSizePool: &sync.Pool{
			New: func() interface{} {
				b := make([]byte, maxBufSize)
				return &b
			},
		},
	}
	for i := int32(0); i < int32(len(layerStartSizeArray)); i++ {
		for j := int32(0); j < oneLayerSize; j++ {
			s := layerStartSizeArray[i] + j*layerStepSizeArray[i]
			buffPool.buffSizeArray[i][j] = s
			buffPool.pools[i][j] = &sync.Pool{
				New: func() interface{} {
					b := make([]byte, s)
					return &b
				},
			}
		}
	}
	return buffPool
}

// BufferPool.Alloc allocate memory with size and return it's pointer
func (bp *BufferPool) Alloc(size int32) *[]byte {
	pool := bp.findPool(size)
	if pool == nil {
		b := make([]byte, size)
		return &b
	}
	bufp := pool.Get().(*[]byte)
	buf := (*bufp)[:size]
	return &buf
}

// BufferPool.Free free memory with bytes pointer
func (bp *BufferPool) Free(buffer *[]byte) {
	pool := bp.findPool(int32(cap(*buffer)))
	if pool == nil {
		return
	}
	pool.Put(buffer)
}

// BufferPool.findPool get pool with size
func (bp *BufferPool) findPool(size int32) *sync.Pool {
	if size > maxBufSize {
		return nil
	}
	if size <= layerStartSizeArray[0] {
		return bp.pools[0][0]
	}
	// layerStartSizeArray[len(layerStartSizeArray)-1]+(OneLayerSize-1)*layerStepSizeArray[len(layerStepSizeArray)-1]
	if size > bp.buffSizeArray[len(layerStartSizeArray)-1][oneLayerSize-1] {
		return bp.maxSizePool
	}

	left, right := int32(0), int32(len(layerStartSizeArray)-1)
	if size > layerStartSizeArray[right] {
		left = right
	} else {
		var mid int32
		for left < right {
			mid = (left + right) / 2
			if mid == left || mid == right {
				break
			}
			if size < layerStartSizeArray[mid] {
				right = mid
			} else if size > layerStartSizeArray[mid] {
				left = mid
			} else {
				return bp.pools[mid][0]
			}
		}
	}
	size -= layerStartSizeArray[left]
	size = (size + layerStepSizeArray[left] - 1) / layerStepSizeArray[left]
	if size >= oneLayerSize {
		return bp.pools[left+1][0]
	}
	return bp.pools[left][size]
}

// default BufferPool
var defaultBuffPool *BufferPool

func init() {
	defaultBuffPool = NewBufferPool()
}

// GetBuffPool get default BufferPool
func GetBuffPool() *BufferPool {
	return defaultBuffPool
}
