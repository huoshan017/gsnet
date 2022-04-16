package msg

import (
	"errors"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/packet"
	"github.com/huoshan017/gsnet/pool"
)

var (
	ErrMsgBodyIncomplete = errors.New("gsnet: message body incomplete")
)

type MsgSession struct {
	sess   common.ISession
	codec  IMsgCodec
	mapper *IdMsgMapper
}

func (s *MsgSession) GetSess() common.ISession {
	return s.sess
}

func (s *MsgSession) GetId() uint64 {
	return s.sess.GetId()
}

func (s *MsgSession) SetData(key string, value any) {
	s.sess.SetData(key, value)
}

func (s *MsgSession) GetData(key string) any {
	return s.sess.GetData(key)
}

func (s *MsgSession) SendMsg(msgid MsgIdType, msg any) error {
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

func (s *MsgSession) SendMsgNoCopy(msgid MsgIdType, msg any) error {
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

func (s *MsgSession) splitIdAndMsg(msgdata []byte) (MsgIdType, any, error) {
	msgid, msgdata, err := splitIdAndMsg(msgdata)
	if err != nil {
		return 0, nil, err
	}
	msgobj := s.mapper.GetReflectNewObject(msgid)
	if msgobj == nil {
		e := ErrMsgIdMapperTypeNotFound(msgid)
		common.CheckAndRegisterNoDisconnectError(e)
		return 0, nil, e
	}
	err = s.codec.Decode(msgdata, msgobj)
	if err != nil {
		return 0, nil, err
	}
	return msgid, msgobj, nil
}

func splitIdAndMsg(data []byte) (msgid MsgIdType, msgdata []byte, err error) {
	if len(data) < 4 {
		return 0, nil, ErrMsgBodyIncomplete
	}
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
