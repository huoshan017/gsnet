package worker

import "errors"

var (
	ErrPacketTypeNotSupported = errors.New("gsnet: packet data type not supported")
)
