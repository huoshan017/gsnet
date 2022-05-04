package msg

import "time"

// MsgIdType message id type
type MsgIdType uint32

// IMsgCodec message codec interface
type IMsgCodec interface {
	Encode(any) ([]byte, error)
	Decode([]byte, any) error
}

// IMsgSessionHandler interface for message event handler
type IMsgSessionEventHandler interface {
	OnConnected(*MsgSession)
	OnReady(*MsgSession)
	OnDisconnected(*MsgSession, error)
	OnTick(*MsgSession, time.Duration)
	OnError(error)
	OnMsgHandle(*MsgSession, MsgIdType, any) error
}
