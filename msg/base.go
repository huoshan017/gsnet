package msg

import "time"

// MsgIdType message id type
type MsgIdType uint32

// IMsgCodec message codec interface
type IMsgCodec interface {
	Encode(interface{}) ([]byte, error)
	Decode([]byte, interface{}) error
}

// IMsgSessionHandler interface for message event handler
type IMsgSessionEventHandler interface {
	OnConnected(*MsgSession)
	OnDisconnected(*MsgSession, error)
	OnTick(*MsgSession, time.Duration)
	OnError(error)
	OnMsgHandle(*MsgSession, MsgIdType, interface{}) error
}
