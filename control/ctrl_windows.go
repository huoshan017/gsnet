package control

import (
	"syscall"

	"golang.org/x/sys/windows"
)

func GetControl(options CtrlOptions) func(network, address string, c syscall.RawConn) error {
	return func(network, address string, c syscall.RawConn) (err error) {
		e := c.Control(func(fd uintptr) {
			if err = windows.SetsockoptInt(windows.Handle(fd), windows.SOL_SOCKET, windows.SO_REUSEADDR, options.ReuseAddr); err != nil {
				return
			}
		})
		if e != nil {
			return e
		}
		return
	}
}
