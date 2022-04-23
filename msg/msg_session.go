package msg

import (
	"errors"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/packet"
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
	pData := pool.GetBuffPool().Alloc(int32(int(s.getHeaderLength()) + len(msgdata)))
	if err = s.formatHeaderTo(*pData, msgid); err != nil {
		return err
	}
	copy((*pData)[s.getHeaderLength():], msgdata[:])
	return s.sess.SendPoolBuffer(pData, packet.MemoryManagementPoolUserManualFree)
}

func (s *MsgSession) SendMsgNoCopy(msgid MsgIdType, msg any) error {
	msgdata, err := s.codec.Encode(msg)
	if err != nil {
		return err
	}
	idHeader := make([]byte, s.getHeaderLength())
	if err = s.formatHeaderTo(idHeader, msgid); err != nil {
		return err
	}
	return s.sess.SendBytesArray([][]byte{idHeader, msgdata}, false)
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
