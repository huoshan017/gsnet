package common

import (
	"errors"
	"sync"
	"sync/atomic"

	"github.com/huoshan017/gsnet/log"
)

var (
	ErrUnlimitedChanClosed = errors.New("gsnet: unlimited chan closed")
)

type unlimitedChan struct {
	inChan      chan wrapperSendData
	outChan     chan wrapperSendData
	closeChan   chan struct{}
	closed      int32
	onceCloseIn sync.Once
}

func newUnlimitedChan() *unlimitedChan {
	uc := &unlimitedChan{
		inChan:    make(chan wrapperSendData),
		outChan:   make(chan wrapperSendData),
		closeChan: make(chan struct{}),
	}
	uc.run()
	return uc
}

func (c *unlimitedChan) run() {
	go func() {
		var sendList *slist
		defer func() {
			sendList.recycle()
			if err := recover(); err != nil {
				log.WithStack(err)
			}
		}()
		var (
			inData, outData wrapperSendData
			ok              bool = true
			outDataChan     chan wrapperSendData
		)
		sendList = newSlist()
		for ok {
			if sendList.getLength() > 0 {
				outData, _ = sendList.peekFront()
				outDataChan = c.outChan
			} else {
				outDataChan = nil
			}
			select {
			case inData, ok = <-c.inChan:
				if !ok {
					continue
				}
				if sendList.getLength() > 0 {
					sendList.pushBack(inData)
				} else {
					select {
					case c.outChan <- inData:
					default:
						sendList.pushBack(inData)
					}
				}
			case outDataChan <- outData:
				sendList.deleteFront()
			case <-c.closeChan:
				ok = false
			}
		}
		close(c.outChan)
	}()
}

func (c *unlimitedChan) PushBack(wd wrapperSendData) bool {
	if atomic.LoadInt32(&c.closed) == 1 {
		c.onceCloseIn.Do(func() {
			close(c.inChan)
		})
		return false
	}
	select {
	case <-c.closeChan:
		c.onceCloseIn.Do(func() {
			close(c.inChan)
		})
		return false
	case c.inChan <- wd:
	}
	return true
}

func (c *unlimitedChan) PopFront() (wrapperSendData, bool) {
	wd, o := <-c.outChan
	if !o {
		return nullWrapperSendData, false
	}
	return wd, true
}

func (c *unlimitedChan) Finalize() {

}

func (c *unlimitedChan) Close() {
	if atomic.LoadInt32(&c.closed) == 1 {
		return
	}
	close(c.closeChan)
	atomic.StoreInt32(&c.closed, 1)
}
