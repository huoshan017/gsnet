package common

import (
	"errors"
	"fmt"
)

var (
	ErrConnClosed    = errors.New("gsnet: connetion is closed")
	ErrSendChanFull  = errors.New("gsnet: send chan full")
	ErrRecvChanEmpty = errors.New("gsnet: recv chan empty")
	ErrCancelWait    = errors.New("gsnet: cancel wait")
	ErrNoMsgHandle   = func(msgid uint32) error {
		return fmt.Errorf("gsnet: no message %v handle", msgid)
	}
	ErrNotImplement = func(funcName string) error {
		return fmt.Errorf("gsnet: not implement %v", funcName)
	}
)

var noDisconnectErrMap = make(map[error]struct{})

func init() {
	noDisconnectErrMap[ErrSendChanFull] = struct{}{}
	noDisconnectErrMap[ErrRecvChanEmpty] = struct{}{}
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
