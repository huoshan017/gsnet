package common

import (
	"errors"
	"fmt"
	"time"

	"github.com/huoshan017/gsnet/log"
	"github.com/huoshan017/gsnet/packet"
)

var (
	ErrBasePacketHandlerStateDismatch = func(state HandlerState) error {
		return fmt.Errorf("base packet handler state %v dismatch", state)
	}
	ErrBasePacketHandlerDisableHeartbeat           = errors.New("base packet handler disable heartbeat")
	ErrBasePacketHandlerClientCantRecvHandshake    = errors.New("base packet handler for client cant receive handshake")
	ErrBasePacketHandlerServerCantRecvHandshakeAck = errors.New("base packet handler for server cant receive handshake ack")
	ErrBasePacketHandlerClientCantRecvHeartbeat    = errors.New("base packet handler for client cant receive heartbeat")
	ErrBasePacketHandlerServerCantRecvHeartbeatAck = errors.New("base packet handler for server cant receive heartbeat ack")
)

type IBasePacketHandler interface {
	OnHandleHandshake(pak packet.IPacket) (int32, error)
	OnPreHandle(packet.IPacket) (int32, error)
	OnPostHandle(packet.IPacket) error
	OnUpdateHandle() error
}

type IPacketEventHandler interface {
	OnHandshakeDone(args ...any) error
}

type HandlerState int32

const (
	HandlerStateNotBegin  HandlerState = iota
	HandlerStateHandshake HandlerState = 1
	HandlerStateNormal    HandlerState = 2
)

type DefaultBasePacketHandler struct {
	cors         bool
	conn         IConn
	eventHandler IPacketEventHandler
	getter       IPacketBuilderArgsGetter
	resend       IResendEventHandler
	options      *Options
	lastTime     time.Time
	state        HandlerState
}

func NewDefaultBasePacketHandler4Client(conn IConn, eventHandler IPacketEventHandler, resend IResendEventHandler, options *Options) *DefaultBasePacketHandler {
	return &DefaultBasePacketHandler{
		cors:         true,
		conn:         conn,
		eventHandler: eventHandler,
		resend:       resend,
		options:      options,
		lastTime:     time.Now(),
		state:        HandlerStateNotBegin,
	}
}

func NewDefaultBasePacketHandler4Server(conn IConn, getter IPacketBuilderArgsGetter, resend IResendEventHandler, options *Options) *DefaultBasePacketHandler {
	return &DefaultBasePacketHandler{
		cors:     false,
		conn:     conn,
		getter:   getter,
		resend:   resend,
		options:  options,
		lastTime: time.Now(),
		state:    HandlerStateNotBegin,
	}
}

func (h *DefaultBasePacketHandler) OnHandleHandshake(pak packet.IPacket) (int32, error) {
	if h.state == HandlerStateNotBegin {
		if !h.cors {
			h.state = HandlerStateHandshake
		}
	}
	if h.state != HandlerStateHandshake {
		return 0, nil
	}
	var (
		res int32
		err error
		typ = pak.Type()
	)
	if typ == packet.PacketHandshake { // server side
		if !h.cors {
			err = h.sendHandshakeAck()
			if err == nil {
				res = 1
			}
		} else {
			err = ErrBasePacketHandlerClientCantRecvHandshake
		}
	} else if typ == packet.PacketHandshakeAck { // client side
		if !h.cors {
			err = ErrBasePacketHandlerServerCantRecvHandshakeAck
		} else {
			h.state = HandlerStateNormal
			data := pak.Data()
			ct := packet.CompressType(data[0])
			et := packet.EncryptionType(data[1])
			h.options.SetPacketCompressType(ct)
			h.options.SetPacketEncryptionType(et)
			l := data[2]
			if l > 0 {
				key := data[3 : 3+l]
				h.options.SetPacketCryptoKey(key)
				if h.eventHandler != nil {
					h.eventHandler.OnHandshakeDone(ct, et, key)
					log.Infof("handshake ack, compress type %v, encryption type %v, crypto key %v", ct, et, key)
				}
			} else {
				h.eventHandler.OnHandshakeDone(ct, et)
				log.Infof("handshake ack, compress type %v, encryption type %v", ct, et)
			}
			res = 2
		}
	}
	return res, err
}

func (h *DefaultBasePacketHandler) OnPreHandle(pak packet.IPacket) (int32, error) {
	var (
		res int32 = 1
		err error
	)
	typ := pak.Type()
	switch typ {
	case packet.PacketHeartbeat: // server receive
		if !h.options.IsUseHeartbeat() {
			err = ErrBasePacketHandlerDisableHeartbeat
			break
		}
		if h.cors {
			err = ErrBasePacketHandlerClientCantRecvHeartbeat
		} else {
			err = h.sendHeartbeatAck()
		}
	case packet.PacketHeartbeatAck: // client receive
		if !h.options.IsUseHeartbeat() {
			err = ErrBasePacketHandlerDisableHeartbeat
			break
		}
		if !h.cors {
			err = ErrBasePacketHandlerServerCantRecvHeartbeatAck
		}
	case packet.PacketSentAck:
		if h.resend != nil {
			res = h.resend.OnAck(pak)
			if res < 0 {
				log.Fatalf("gsnet: length of rend list less than ack num")
				err = ErrResendDataInvalid
			}
		}
	default:
		// reset heartbeat timer
		if h.options.IsUseHeartbeat() {
			h.lastTime = time.Now()
		}
		res = 0
	}
	return res, err
}

func (h *DefaultBasePacketHandler) OnPostHandle(pak packet.IPacket) error {
	var err error
	if h.resend != nil {
		h.resend.OnProcessed(1)
	}
	return err
}

func (h *DefaultBasePacketHandler) OnUpdateHandle() error {
	var err error
	switch h.state {
	case HandlerStateNotBegin:
		if h.cors {
			err = h.sendHandshake()
			if err == nil {
				h.state = HandlerStateHandshake
			}
		}
	case HandlerStateNormal:
		if h.cors && h.options.IsUseHeartbeat() {
			// heartbeat timeout to disconnect
			disconnectTimeout := h.options.GetDisconnectHeartbeatTimeout()
			if disconnectTimeout <= 0 {
				disconnectTimeout = DefaultDisconnectHeartbeatTimeout
			}
			duration := time.Since(h.lastTime)
			if duration >= disconnectTimeout {
				h.conn.Close()
				return err
			}

			// heartbeat timespan
			minSpan := h.options.GetMinHeartbeatTimeSpan()
			if minSpan < DefaultMinimumHeartbeatTimeSpan {
				minSpan = DefaultMinimumHeartbeatTimeSpan
			}
			span := h.options.GetHeartbeatTimeSpan()
			if span <= 0 {
				span = DefaultHeartbeatTimeSpan
			} else if span < minSpan {
				span = minSpan
			}

			// send heartbeat
			if duration >= span {
				err = h.sendHeartbeat()
				if err != nil {
					h.lastTime = time.Now()
				}
			}
		}
	}
	if h.resend != nil {
		err = h.resend.OnUpdate(h.conn)
	}
	return err
}

func (h *DefaultBasePacketHandler) sendHandshake() error {
	return h.conn.Send(packet.PacketHandshake, []byte{}, false)
}

func (h *DefaultBasePacketHandler) sendHandshakeAck() error {
	if h.getter == nil {
		return nil
	}
	ct := h.options.GetPacketCompressType()
	et := h.options.GetPacketEncryptionType()
	args := h.getter.Get()
	if len(args) < 1 {
		return errors.New("gsnet: packet builder arguments not enough")
	}
	key, o := args[0].([]byte)
	if !o {
		return errors.New("gsnet: packet builder argument crypto key type cast failed")
	}
	log.Infof("send compress type %v, encryption type %v, key %v", ct, et, key)
	data := []byte{
		byte(ct),       // compress type
		byte(et),       // encryption type
		byte(len(key)), // crypto key
	}
	data = append(data, key...)
	return h.conn.Send(packet.PacketHandshakeAck, data, false)
}

func (h *DefaultBasePacketHandler) sendHeartbeat() error {
	return h.conn.Send(packet.PacketHeartbeat, []byte{}, false)
}

func (h *DefaultBasePacketHandler) sendHeartbeatAck() error {
	return h.conn.Send(packet.PacketHeartbeatAck, []byte{}, false)
}
