package common

import (
	"github.com/huoshan017/gsnet/packet"
	"github.com/huoshan017/gsnet/pool"
)

type SessionChannel struct {
	id   uint64
	sess ISession
}

func NewSessionChannel(id uint64, sess ISession) *SessionChannel {
	return &SessionChannel{
		id:   id,
		sess: sess,
	}
}

func (sc *SessionChannel) GetId() uint64 {
	return sc.sess.GetId()
}

func (sc *SessionChannel) send(data []byte) error {
	temp := make([]byte, 8)
	Uint64ToBuffer(sc.id, temp)
	return sc.sess.SendBytesArray([][]byte{temp, data}, false)
}

func (sc *SessionChannel) sendOnCopy(data []byte) error {
	buffer := pool.GetBuffPool().Alloc(8 + int32(len(data)))
	Uint64ToBuffer(sc.id, (*buffer)[:8])
	copy((*buffer)[8:], data)
	return sc.sess.SendPoolBuffer(buffer)
}

func (sc *SessionChannel) Send(data []byte, isCopy bool) error {
	if !isCopy {
		return sc.send(data)
	} else {
		return sc.sendOnCopy(data)
	}
}

func (sc *SessionChannel) SendBytesArray(datas [][]byte, isCopy bool) error {
	temp := make([]byte, 8)
	Uint64ToBuffer(sc.id, temp)
	datas = append([][]byte{temp}, datas...)
	return sc.sess.SendBytesArray(datas, isCopy)
}

func (sc *SessionChannel) SendPoolBuffer(buffer *[]byte) error {
	buf := pool.GetBuffPool().Alloc(8)
	Uint64ToBuffer(sc.id, (*buf)[:])
	return sc.sess.SendPoolBufferArray([]*[]byte{buf, buffer})
}

func (sc *SessionChannel) SendPoolBufferArray(bufferArray []*[]byte) error {
	buf := pool.GetBuffPool().Alloc(8)
	Uint64ToBuffer(sc.id, (*buf)[:])
	bufferArray = append([]*[]byte{buf}, bufferArray...)
	return sc.sess.SendPoolBufferArray(bufferArray)
}

func (sc *SessionChannel) Close() {
	sc.sess.Close()
}

func (sc *SessionChannel) CloseWaitSecs(secs int) {
	sc.sess.CloseWaitSecs(secs)
}

func (sc *SessionChannel) AddInboundHandle(id int32, handle func(ISession, packet.IPacket) error) {
	sc.sess.AddInboundHandle(id, handle)
}

func (sc *SessionChannel) GetInboundHandles() map[int32]func(ISession, packet.IPacket) error {
	return sc.sess.GetInboundHandles()
}

func (sc *SessionChannel) GetPacketChannel() chan IdWithPacket {
	return sc.sess.GetPacketChannel()
}

func (sc *SessionChannel) SetUserData(key string, data any) {
	sc.sess.SetUserData(key, data)
}

func (sc *SessionChannel) GetUserData(key string) any {
	return sc.sess.GetUserData(key)
}
