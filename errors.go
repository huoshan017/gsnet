package gsnet

import (
	"errors"
)

var ErrBodyLenInvalid = errors.New("gsnet: receive body len too large")
var ErrConnClosed = errors.New("gsnet: conn is closed")
var ErrSendChanFull = errors.New("gsnet: send chan full")
var ErrRecvChanEmpty = errors.New("gsnet: recv chan empty")
var ErrNoMsgHandle = errors.New("gsnet: no message handle")

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
