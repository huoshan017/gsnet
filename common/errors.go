package common

import (
	"errors"
	"fmt"
)

var (
	ErrNoError             = errors.New("no error")
	ErrUnknownNetwork      = errors.New("unknown network")
	ErrConnClosed          = errors.New("connetion is closed")
	ErrSendListFull        = errors.New("send list full")
	ErrRecvListEmpty       = errors.New("recv list empty")
	ErrCancelWait          = errors.New("cancel wait")
	ErrResendDisable       = errors.New("resend disable")
	ErrResendDataInvalid   = errors.New("resend data invalid")
	ErrSentPacketCacheFull = errors.New("sent packet cache is full")
	ErrNoMsgHandle         = func(msgid uint32) error {
		return fmt.Errorf("no message %v handle", msgid)
	}
	ErrNotImplement = func(funcName string) error {
		return fmt.Errorf("not implement %v", funcName)
	}
	ErrPacketTypeNotSupported = errors.New("gsnet: packet data type not supported")
)

var noDisconnectErrMap = make(map[error]struct{})

func init() {
	noDisconnectErrMap[ErrSendListFull] = struct{}{}
	noDisconnectErrMap[ErrRecvListEmpty] = struct{}{}
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
