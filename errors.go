package gsnet

import (
	"errors"
)

var (
	ErrBodyLenInvalid = errors.New("gsnet: receive body length too long")
	ErrConnClosed     = errors.New("gsnet: connetion is closed")
	ErrSendChanFull   = errors.New("gsnet: send chan full")
	ErrRecvChanEmpty  = errors.New("gsnet: recv chan empty")
	ErrNoMsgHandle    = errors.New("gsnet: no message handle")
)

var noDisconnectErrMap = make(map[error]struct{})

func init() {
	noDisconnectErrMap[ErrSendChanFull] = struct{}{}
	noDisconnectErrMap[ErrRecvChanEmpty] = struct{}{}
	noDisconnectErrMap[ErrNoMsgHandle] = struct{}{}
}

func IsNoDisconnectError(err error) bool {
	_, o := noDisconnectErrMap[err]
	return o
}

func RegisterNoDisconnectError(err error) {
	noDisconnectErrMap[err] = struct{}{}
}
