package control

import (
	"syscall"

	"golang.org/x/sys/unix"
)

func GetControl(options CtrlOptions) func(network, address string, c syscall.RawConn) error {
	return func(network, address string, c syscall.RawConn) (err error) {
		e := c.Control(func(fd uintptr) {
			// SO_REUSEADDR
			if err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, options.ReuseAddr); err != nil {
				panic(err)
			}
			// SO_REUSEPORT
			if err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, options.ReusePort); err != nil {
				panic(err)
			}
		})
		if e != nil {
			return e
		}
		return
	}
}
