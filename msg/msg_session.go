package msg

import (
	"errors"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/pool"
)

const (
	DefaultMsgHeaderLength = 4
)

var (
	ErrMsgBodyIncomplete               = errors.New("gsnet: message body incomplete")
	ErrMsgHeaderFormatNeedLargerBuffer = errors.New("gsnet: message header format need bigger buffer")
)

type MsgSession struct {
	sess    common.ISession
	codec   IMsgCodec
	mapper  *IdMsgMapper
	options *MsgOptions
}

func (s *MsgSession) GetSess() common.ISession {
	return s.sess
}

func (s *MsgSession) GetId() uint64 {
	return s.sess.GetId()
}

func (s *MsgSession) SetUserData(key string, value any) {
	s.sess.SetUserData(key, value)
}

func (s *MsgSession) GetData(key string) any {
	return s.sess.GetUserData(key)
}

func (s *MsgSession) serializeMsgOnCopy(msgid MsgIdType, msg any) (*[]byte, error) {
	msgdata, err := s.codec.Encode(msg)
	if err != nil {
		return nil, err
	}
	pData := pool.GetBuffPool().Alloc(int32(int(s.getHeaderLength()) + len(msgdata)))
	if err = s.formatHeaderTo(*pData, msgid); err != nil {
		return nil, err
	}
	copy((*pData)[s.getHeaderLength():], msgdata[:])
	return pData, nil
}

func (s *MsgSession) SendMsgOnCopy(msgid MsgIdType, msg any) error {
	pData, err := s.serializeMsgOnCopy(msgid, msg)
	if err != nil {
		return err
	}
	return s.sess.SendPoolBuffer(pData)
}

func (s *MsgSession) serializeMsg(msgid MsgIdType, msg any) ([][]byte, error) {
	msgdata, err := s.codec.Encode(msg)
	if err != nil {
		return nil, err
	}
	idHeader := make([]byte, s.getHeaderLength())
	if err = s.formatHeaderTo(idHeader, msgid); err != nil {
		return nil, err
	}
	return [][]byte{idHeader, msgdata}, nil
}

func (s *MsgSession) SendMsg(msgid MsgIdType, msg any) error {
	dataArray, err := s.serializeMsg(msgid, msg)
	if err != nil {
		return err
	}
	return s.sess.SendBytesArray(dataArray, false)
}

func (s *MsgSession) Close() {
	s.sess.Close()
}

func (s *MsgSession) CloseWait(secs int) {
	s.sess.CloseWaitSecs(secs)
}

func (s *MsgSession) splitIdAndMsg(msgdata []byte) (MsgIdType, any, error) {
	msgid, err := s.unformatHeaderFrom(msgdata)
	if err != nil {
		return 0, nil, err
	}
	msgobj := s.mapper.GetReflectNewObject(msgid)
	if msgobj == nil {
		e := ErrMsgIdMapperTypeNotFound(msgid)
		common.CheckAndRegisterNoDisconnectError(e)
		return 0, nil, e
	}
	msgdata = msgdata[s.getHeaderLength():]
	err = s.codec.Decode(msgdata, msgobj)
	if err != nil {
		return 0, nil, err
	}
	return msgid, msgobj, nil
}

func (s *MsgSession) formatHeaderTo(data []byte, msgid MsgIdType) error {
	var err error
	if s.options.GetHeaderFormatFunc() != nil {
		err = s.options.GetHeaderFormatFunc()(msgid, data)
	} else {
		err = DefaultMsgHeaderFormat(msgid, data) // thread safe
	}
	return err
}

func (s *MsgSession) unformatHeaderFrom(data []byte) (MsgIdType, error) {
	var (
		msgid MsgIdType
		err   error
	)
	if s.options.GetHeaderUnformatFunc() != nil {
		msgid, err = s.options.GetHeaderUnformatFunc()(data)
	} else {
		msgid, err = DefaultMsgHeaderUnformat(data)
	}
	return msgid, err
}

func (s *MsgSession) getHeaderLength() uint8 {
	length := s.options.GetHeaderLength()
	if length == 0 {
		length = DefaultMsgHeaderLength
	}
	return length
}
