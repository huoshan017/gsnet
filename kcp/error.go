package kcp

import "errors"

var (
	ErrNeedSynAck = errors.New("gsnet: kcp connection need frame synack")
	ErrConvToken  = errors.New("gsnet: kcp connection conversation or token dismatch")
)
