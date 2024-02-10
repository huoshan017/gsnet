package handler

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/log"
	"github.com/huoshan017/gsnet/options"
	"github.com/huoshan017/gsnet/packet"
	"github.com/huoshan017/gsnet/protocol"
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

type roleType int8

const (
	roleTypeClient roleType = iota
	roleTypeServer roleType = 1
)

type IBasePacketHandler interface {
	OnStart(any) error
	OnHandleHandshake(pak packet.IPacket) (int32, error)
	OnHandleReconnect(pak packet.IPacket) (int32, error)
	OnPreHandle(packet.IPacket) (int32, error)
	OnPostHandle(packet.IPacket)
	OnUpdateHandle() error
}

type IPacketEventHandler interface {
	OnHandshakeDone(args ...any) error
}

type IPacketArgsGetter interface {
	Get() []any
}

type HandlerState int32

const (
	HandlerStateNotBegin  HandlerState = iota
	HandlerStateHandshake HandlerState = 1
	HandlerStateReconnect HandlerState = 2
	HandlerStateNormal    HandlerState = 10
)

const (
	ReconnectStateNone      = 0
	ReconnectStateSyn       = 1
	ReconnectStateAck       = 2
	ReconnectStateTransport = 3
	ReconnectStateEnd       = 4
)

const (
	HandshakeStateNone        = 0
	HandshakeStateServerReady = 1
	HandshakeStateClientReady = 2
)

type DefaultBasePacketHandler struct {
	rtype              roleType
	sess               common.ISession
	conn               common.IConn
	packetEventHandler IPacketEventHandler
	argsGetter         IPacketArgsGetter
	resendEventHandler common.IResendEventHandler
	options            *options.Options
	lastTime           time.Time
	state              HandlerState
	reconnInfo         *common.ReconnectInfo
	reconnected        bool
	reconnInfoMap      sync.Map
}

func NewDefaultBasePacketHandler4Client(
	sess common.ISession,
	packetEventHandler IPacketEventHandler,
	resendEventHandler common.IResendEventHandler,
	options *options.Options) *DefaultBasePacketHandler {
	var conn common.IConn
	connGetter, o := sess.(common.IConnGetter)
	if o {
		conn = connGetter.Conn()
	}
	return &DefaultBasePacketHandler{
		rtype:              roleTypeClient,
		sess:               sess,
		conn:               conn,
		packetEventHandler: packetEventHandler,
		resendEventHandler: resendEventHandler,
		options:            options,
		lastTime:           time.Now(),
		state:              HandlerStateNotBegin,
	}
}

func NewDefaultBasePacketHandler4Server(
	sess common.ISession,
	argsGetter IPacketArgsGetter,
	resendEventHandler common.IResendEventHandler,
	options *options.Options) *DefaultBasePacketHandler {
	var conn common.IConn
	connGetter, o := sess.(common.IConnGetter)
	if o {
		conn = connGetter.Conn()
	}
	return &DefaultBasePacketHandler{
		rtype:              roleTypeServer,
		sess:               sess,
		conn:               conn,
		argsGetter:         argsGetter,
		resendEventHandler: resendEventHandler,
		options:            options,
		lastTime:           time.Now(),
		state:              HandlerStateNotBegin,
	}
}

func (h *DefaultBasePacketHandler) OnStart(d any) error {
	if h.state != HandlerStateNotBegin {
		return nil
	}

	var (
		err error
	)

	if d != nil {
		h.reconnInfo, _ = d.(*common.ReconnectInfo)
	}

	if h.rtype == roleTypeClient {
		// handle reconnect
		if h.reconnInfo != nil {
			h.state = HandlerStateReconnect
			if err = h.sendReconnectSyn(); err != nil {
				return err
			}
		}

		if h.reconnInfo == nil || h.reconnected {
			// handle handshake
			err = h.sendHandshake()
			if err == nil {
				h.state = HandlerStateHandshake
			}
		}
	}

	return err
}

func (h *DefaultBasePacketHandler) OnHandleReconnect(pak packet.IPacket) (int32, error) {
	var (
		res int32
		err error
	)
	if h.state != HandlerStateReconnect {
		return ReconnectStateNone, nil
	}
	switch pak.Type() {
	case packet.PacketReconnectSyn:
		if h.rtype == roleTypeServer {
			var reconnSyn protocol.ReconnectSyn
			if err = proto.Unmarshal(pak.Data(), &reconnSyn); err == nil {
				reconnSyn.GetCurrSessionId()
			}
		}
	case packet.PacketReconnectAck:
	case packet.PacketReconnectTransport:
	case packet.PacketReconnectEnd:
	}
	return res, err
}

func (h *DefaultBasePacketHandler) OnHandleHandshake(pak packet.IPacket) (int32, error) {
	if h.state == HandlerStateNotBegin {
		if h.rtype == roleTypeServer {
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
		if h.rtype == roleTypeServer {
			err = h.sendHandshakeAck()
			if err == nil {
				res = HandshakeStateServerReady // server ready
			}
		} else {
			err = ErrBasePacketHandlerClientCantRecvHandshake
		}
	} else if typ == packet.PacketHandshakeAck { // client side
		if h.rtype == roleTypeServer {
			err = ErrBasePacketHandlerServerCantRecvHandshakeAck
		} else {
			h.state = HandlerStateNormal
			var hd protocol.HandshakeData
			if err = proto.Unmarshal(pak.Data(), &hd); err != nil {
				return res, err
			}
			h.options.SetPacketCompressType(packet.CompressType(hd.CompressType))
			h.options.SetPacketEncryptionType(packet.EncryptionType(hd.EncryptionType))
			if h.packetEventHandler != nil {
				err = h.packetEventHandler.OnHandshakeDone(packet.CompressType(hd.CompressType), packet.EncryptionType(hd.EncryptionType), hd.EncryptionKey, hd.SessionId)
				log.Infof("handshake ack, compress type %v, encryption type %v, crypto key %v, sessoion id %v",
					hd.CompressType, hd.EncryptionType, hd.EncryptionKey, hd.SessionId)
			}
			if err == nil {
				res = HandshakeStateClientReady // client ready
			}
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
		if h.rtype == roleTypeClient {
			err = ErrBasePacketHandlerClientCantRecvHeartbeat
		} else {
			err = h.sendHeartbeatAck()
		}
	case packet.PacketHeartbeatAck: // client receive
		if !h.options.IsUseHeartbeat() {
			err = ErrBasePacketHandlerDisableHeartbeat
			break
		}
		if h.rtype == roleTypeServer {
			err = ErrBasePacketHandlerServerCantRecvHeartbeatAck
		}
	case packet.PacketSentAck:
		if h.resendEventHandler != nil {
			err = h.resendEventHandler.OnAck(pak)
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

func (h *DefaultBasePacketHandler) OnPostHandle(pak packet.IPacket) {
	if h.resendEventHandler != nil {
		h.resendEventHandler.OnProcessed(pak)
	}
}

func (h *DefaultBasePacketHandler) OnUpdateHandle() error {
	var err error
	switch h.state {

	case HandlerStateNormal:
		if h.rtype == roleTypeClient && h.options.IsUseHeartbeat() {
			// heartbeat timeout to disconnect
			disconnectTimeout := h.options.GetDisconnectHeartbeatTimeout()
			if disconnectTimeout <= 0 {
				disconnectTimeout = options.DefaultDisconnectHeartbeatTimeout
			}
			duration := time.Since(h.lastTime)
			if duration >= disconnectTimeout {
				h.sess.Close()
				return err
			}

			// heartbeat timespan
			minSpan := h.options.GetMinHeartbeatTimeSpan()
			if minSpan < options.DefaultMinimumHeartbeatTimeSpan {
				minSpan = options.DefaultMinimumHeartbeatTimeSpan
			}
			span := h.options.GetHeartbeatTimeSpan()
			if span <= 0 {
				span = options.DefaultHeartbeatTimeSpan
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
	if h.resendEventHandler != nil {
		err = h.resendEventHandler.OnUpdate(h.conn)
	}
	return err
}

func (h *DefaultBasePacketHandler) sendReconnectSyn() error {
	return nil
}

func (h *DefaultBasePacketHandler) sendReconnectAck() error {
	return nil
}

func (h *DefaultBasePacketHandler) sendReconnectTransportData() error {
	return nil
}

func (h *DefaultBasePacketHandler) sendReconnectEnd() error {
	return nil
}

func (h *DefaultBasePacketHandler) sendHandshake() error {
	connGetter, o := h.sess.(common.IConnGetter)
	if !o {
		return nil
	}
	return connGetter.Conn().Send(packet.PacketHandshake, []byte{}, false)
}

func (h *DefaultBasePacketHandler) sendHandshakeAck() error {
	if h.argsGetter == nil {
		return nil
	}
	ct := h.options.GetPacketCompressType()
	et := h.options.GetPacketEncryptionType()
	args := h.argsGetter.Get()
	if len(args) < 1 {
		return errors.New("gsnet: packet builder arguments not enough")
	}
	key, o := args[0].([]byte)
	if !o {
		return errors.New("gsnet: packet builder argument crypto key type cast failed")
	}
	//log.Infof("send compress type %v, encryption type %v, key %v", ct, et, key)
	hd := &protocol.HandshakeData{
		CompressType:   int32(ct),
		EncryptionType: int32(et),
		EncryptionKey:  key,
		SessionId:      h.sess.GetId(),
	}
	data, err := proto.Marshal(hd)
	if err != nil {
		return err
	}
	return h.conn.Send(packet.PacketHandshakeAck, data, false)
}

func (h *DefaultBasePacketHandler) sendHeartbeat() error {
	return h.conn.Send(packet.PacketHeartbeat, []byte{}, false)
}

func (h *DefaultBasePacketHandler) sendHeartbeatAck() error {
	return h.conn.Send(packet.PacketHeartbeatAck, []byte{}, false)
}
