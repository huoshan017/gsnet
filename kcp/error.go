package kcp

import "errors"

var (
	ErrNeedSynAck        = errors.New("gsnet: kcp connection need frame synack")
	ErrDecodeFrameFailed = errors.New("gsnet: kcp connection decode frame failed")
	ErrUDPConvToken      = errors.New("gsnet: kcp connection conversation or token dismatch")
)
