package gsnet

import (
	"errors"
)

var ErrConnClosed = errors.New("netlib: conn is closed")
var ErrSendChanFull = errors.New("netlib: send chan full")
var ErrRecvChanEmpty = errors.New("netlib: recv chan empty")
var ErrNoMsgHandle = errors.New("netlib: no message handle")

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
