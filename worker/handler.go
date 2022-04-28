package worker

import (
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/pool"
)

type commonHandler struct {
	connectHandle    func(common.ISession)
	disconnectHandle func(common.ISession, error)
	tickHandle       func(common.ISession, time.Duration)
	errorHandle      func(error)
}

func (h *commonHandler) setConnectHandle(handle func(common.ISession)) {
	h.connectHandle = handle
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
	//log.Infof("gsnet: worker client %v connected server", h.owner.id)
	if h.connectHandle != nil {
		h.connectHandle(sess)
	}
}

func (h *commonHandler) OnTick(sess common.ISession, tick time.Duration) {
	if h.tickHandle != nil {
		h.tickHandle(sess, tick)
	}
}

func (h *commonHandler) OnDisconnect(sess common.ISession, err error) {
	//log.Infof("gsnet: worker client %v disconnect from server, err %v", h.owner.id, err)
	if h.disconnectHandle != nil {
		h.disconnectHandle(sess, err)
	}
}

func (h *commonHandler) OnError(err error) {
	//log.Infof("gsnet: worker client %v occur err %v", err)
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
