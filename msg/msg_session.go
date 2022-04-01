package msg

import (
	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/common/packet"
	"github.com/huoshan017/gsnet/common/pool"
)

type MsgSession struct {
	sess   common.ISession
	codec  IMsgCodec
	mapper *IdMsgMapper
}

func (s *MsgSession) GetId() uint64 {
	return s.sess.GetId()
}

func (s *MsgSession) SetData(key string, value interface{}) {
	s.sess.SetData(key, value)
}

func (s *MsgSession) GetData(key string) interface{} {
	return s.sess.GetData(key)
}

func (s *MsgSession) SendMsg(msgid MsgIdType, msg interface{}) error {
	msgdata, err := s.codec.Encode(msg)
	if err != nil {
		return err
	}
	//data := make([]byte, 4+len(msgdata))
	pData := pool.GetBuffPool().Alloc(int32(4 + len(msgdata)))
	genMsgIdHeader(msgid, *pData)
	copy((*pData)[4:], msgdata[:])
	return s.sess.SendPoolBuffer(pData, packet.MemoryManagementPoolUserManualFree)
}

func (s *MsgSession) SendMsgNoCopy(msgid MsgIdType, msg interface{}) error {
	msgdata, err := s.codec.Encode(msg)
	if err != nil {
		return err
	}
	idHeader := make([]byte, 4)
	genMsgIdHeader(msgid, idHeader)
	return s.sess.SendBytesArray([][]byte{idHeader, msgdata}, false)
}

func (s *MsgSession) Close() {
	s.sess.Close()
}

func (s *MsgSession) CloseWait(secs int) {
	s.sess.CloseWaitSecs(secs)
}

func (s *MsgSession) splitIdAndMsg(msgdata []byte) (MsgIdType, interface{}, error) {
	msgid, msgdata := splitIdAndMsg(msgdata)
	msgobj := s.mapper.GetReflectNewObject(msgid)
	if msgobj == nil {
		return 0, nil, ErrMsgIdMapperTypeNotFound
	}
	err := s.codec.Decode(msgdata, msgobj)
	if err != nil {
		return 0, nil, err
	}
	return msgid, msgobj, nil
}

func splitIdAndMsg(data []byte) (msgid MsgIdType, msgdata []byte) {
	for i := 0; i < 4; i++ {
		msgid += MsgIdType(data[i] << (8 * (4 - i - 1)))
	}
	msgdata = data[4:]
	return
}

func genMsgIdHeader(msgid MsgIdType, idHeader []byte) {
	for i := 0; i < 4; i++ {
		idHeader[i] = byte(msgid >> (8 * (4 - i - 1)))
	}
}
