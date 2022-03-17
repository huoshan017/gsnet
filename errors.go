package gsnet

import (
	"errors"
	"fmt"
)

var (
	ErrBodyLenInvalid  = errors.New("gsnet: receive body length too long")
	ErrConnClosed      = errors.New("gsnet: connetion is closed")
	ErrSendChanFull    = errors.New("gsnet: send chan full")
	ErrRecvChanEmpty   = errors.New("gsnet: recv chan empty")
	ErrNoMsgHandle     = errors.New("gsnet: no message handle")
	ErrNoMsgHandleFunc = func(msgid uint32) error {
		return fmt.Errorf("gsnet: no message %v handle", msgid)
	}
	ErrCancelWait = errors.New("gsnet: cancel wait")
)

var noDisconnectErrMap = make(map[error]struct{})

func init() {
	noDisconnectErrMap[ErrSendChanFull] = struct{}{}
	noDisconnectErrMap[ErrRecvChanEmpty] = struct{}{}
	noDisconnectErrMap[ErrNoMsgHandle] = struct{}{}
}

func RegisterNoDisconnectError(err error) {
	noDisconnectErrMap[err] = struct{}{}
}

func IsNoDisconnectError(err error) bool {
	_, o := noDisconnectErrMap[err]
	return o
}

func CheckAndRegisterNoDisconnectError(err error) {
	_, o := noDisconnectErrMap[err]
	if !o {
		noDisconnectErrMap[err] = struct{}{}
	}
}
