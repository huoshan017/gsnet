package client

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/handler"
	"github.com/huoshan017/gsnet/packet"
)

var (
	ErrClientRunUpdateMode   = errors.New("gsnet: client run update mode")
	ErrClientRunMainLoopMode = errors.New("gsnet: client run main loop mode")
	ErrClientNotConnect      = errors.New("gsnet: client not connect")
	ErrClientNotReady        = errors.New("gsnet: client not ready")
	ErrClientConnecting      = errors.New("gsnet: client is connecting")
	ErrClientDisconnecting   = errors.New("gsnet: client is disconnecting")
	ErrClientDisconnected    = errors.New("gsnet: client is disconnected")
)

// 数据客户端
type Client struct {
	address           string
	connTimeout       time.Duration
	connAsyncCallback func(error)
	connector         *Connector
	conn              common.IConn
	sess              *common.SessionEx
	handler           common.ISessionEventHandler
	basePacketHandler handler.IBasePacketHandler
	packetBuilder     *common.PacketBuilder
	resend            *common.ResendData
	options           ClientOptions
	lastTime          time.Time
	ctx               context.Context
	cancel            context.CancelFunc
	isReady           int32
	activeClosed      int32
}

func NewClient(handler common.ISessionEventHandler, options ...common.Option) *Client {
	c := &Client{
		handler: handler,
		options: ClientOptions{Options: *common.NewOptions()},
	}
	for _, option := range options {
		option(&c.options.Options)
	}
	c.init()
	return c
}

func NewClientWithOptions(handler common.ISessionEventHandler, options *ClientOptions) *Client {
	c := &Client{
		options: *options,
		handler: handler,
	}
	c.init()
	return c
}

func (c *Client) init() {
	c.ctx, c.cancel = context.WithCancel(context.Background())
	if c.options.GetPacketPool() == nil {
		c.options.SetPacketPool(packet.GetDefaultPacketPool())
	}
}

func (c *Client) reset() {
	c.connector.Reset()
	c.lastTime = time.Now()
	atomic.StoreInt32(&c.isReady, 0)
	atomic.StoreInt32(&c.activeClosed, 0)
}

func (c *Client) Connect(addr string) error {
	connector := c.newConnector()
	conn, err := connector.Connect(addr)
	if err == nil {
		err = c.doConnectResult(conn)
	}
	c.address = addr
	return err
}

func (c *Client) ConnectWithTimeout(addr string, timeout time.Duration) error {
	connector := c.newConnector()
	conn, err := connector.ConnectWithTimeout(addr, timeout)
	if err == nil {
		err = c.doConnectResult(conn)
	}
	c.address = addr
	c.connTimeout = timeout
	return err
}

func (c *Client) ConnectAsync(addr string, timeout time.Duration, callback func(error)) {
	connector := c.newConnector()
	connector.ConnectAsync(addr, timeout, func(err error) {
		if err == nil {
			err = c.doConnectResult(connector.GetConn())
		}
		if callback != nil {
			callback(err)
		}
	})
	c.address = addr
	c.connTimeout = timeout
	c.connAsyncCallback = callback
}

func (c *Client) newConnector() *Connector {
	c.connector = NewConnector(&c.options.Options)
	return c.connector
}

func (c *Client) doConnectResult(con net.Conn) error {
	var (
		resend *common.ResendData
	)

	switch c.options.GetConnDataType() {
	case 1:
		c.conn = common.NewSimpleConn(con, c.options.Options)
	default:
		c.packetBuilder = common.NewPacketBuilder(&c.options.Options)
		resendConfig := c.options.GetResendConfig()
		if resendConfig != nil {
			resend = common.NewResendData(resendConfig)
		}
		if resend != nil {
			c.conn = common.NewConnUseResend(con, c.packetBuilder, resend, &c.options.Options)
		} else {
			c.conn = common.NewConn(con, c.packetBuilder, &c.options.Options)
		}
	}

	c.sess = common.NewSessionNoId(c.conn)

	// 包事件处理器
	var pakEvtHandler handler.IPacketEventHandler
	if c.packetBuilder != nil {
		pakEvtHandler = &packetEventHandler{builder: c.packetBuilder, sess: c.sess}
	}

	// 重传事件处理器
	var resendEventHandler common.IResendEventHandler
	if c.resend != nil {
		resendEventHandler = resend
	}

	// 基础包处理器
	c.basePacketHandler = handler.NewDefaultBasePacketHandler4Client(c.sess, pakEvtHandler, resendEventHandler, &c.options.Options)

	// 连接跑起来
	c.conn.Run()

	// update模式下先把握手处理掉
	if c.options.GetRunMode() == RunModeOnlyUpdate {
		return c.fromConnect2Ready()
	}

	return nil
}

func (c *Client) Send(data []byte, copyData bool) error {
	if err := c.getError(); err != nil {
		return err
	}
	return c.sess.Send(data, copyData)
}

func (c *Client) SendPoolBuffer(buffer *[]byte) error {
	if err := c.getError(); err != nil {
		return err
	}
	return c.sess.SendPoolBuffer(buffer)
}

func (c *Client) SendBytesArray(bytesArray [][]byte, copyData bool) error {
	if err := c.getError(); err != nil {
		return err
	}
	return c.sess.SendBytesArray(bytesArray, copyData)
}

func (c *Client) SendPoolBufferArray(bufferArray []*[]byte) error {
	if err := c.getError(); err != nil {
		return err
	}
	return c.sess.SendPoolBufferArray(bufferArray)
}

func (c *Client) Update() error {
	if c.options.GetRunMode() != RunModeOnlyUpdate {
		return ErrClientRunUpdateMode
	}

	// 还未连接
	if c.IsNotConnect() || c.IsDisconnected() {
		return nil
	}

	var err error

	// 连接状态
	if c.IsConnecting() {
		_, err = c.connector.WaitResult(false)
		return err
	}

	err = c.handle(1)
	return c.handleErr(err)
}

func (c *Client) Run() {
	if c.options.GetRunMode() != RunModeAsMainLoop {
		c.handler.OnError(ErrClientRunMainLoopMode)
		return
	}

	var (
		err            error
		lastCheck      time.Time = time.Now()
		reconnectTimer *time.Timer
	)

	defer func() {
		if reconnectTimer != nil {
			reconnectTimer.Stop()
		}
	}()

MainLoop:

	if c.IsConnecting() {
		_, err = c.connector.WaitResult(true)
	}

	if err == nil {
		err = c.fromConnect2Ready()
		for err == nil {
			err = c.handle(0)
		}
	}

	c.handleErr(err)

	if c.options.IsReconnect() && atomic.LoadInt32(&c.activeClosed) == 0 {
		c.handleReconnect(&lastCheck, reconnectTimer)
		goto MainLoop
	}
}

func (c *Client) Quit() {
	c.cancel()
}

func (c *Client) Close() {
	if c.options.GetConnCloseWaitSecs() > 0 {
		c.conn.CloseWait(c.options.GetConnCloseWaitSecs())
	} else {
		c.conn.Close()
	}
	atomic.StoreInt32(&c.activeClosed, 1)
}

func (c *Client) CloseWait(secs int) {
	c.conn.CloseWait(secs)
	atomic.StoreInt32(&c.activeClosed, 1)
}

func (c *Client) GetSession() common.ISession {
	return c.sess
}

func (c *Client) IsNotConnect() bool {
	if c.connector == nil {
		return true
	}
	return c.connector.IsNotConnect()
}

func (c *Client) IsConnecting() bool {
	if c.connector == nil {
		return false
	}
	return c.connector.IsConnecting()
}

func (c *Client) IsConnected() bool {
	if c.connector == nil {
		return false
	}
	return c.connector.IsConnected()
}

func (c *Client) IsReady() bool {
	return atomic.LoadInt32(&c.isReady) > 0
}

func (c *Client) IsDisconnected() bool {
	if c.connector == nil {
		return false
	}
	return c.connector.IsDisconnected()
}

func (c *Client) IsDisconnecting() bool {
	if c.connector == nil {
		return false
	}
	return c.connector.IsDisconnecting()
}

func (c *Client) fromConnect2Ready() error {
	var (
		res bool
		err error
	)
	c.handler.OnConnect(c.sess)
	for {
		res, err = c.handleHandshake(0)
		if err != nil || res {
			break
		}
	}
	if err == nil {
		c.handler.OnReady(c.sess)
		atomic.StoreInt32(&c.isReady, 1)
	}
	return err
}

func (c *Client) handleHandshake(mode int32) (bool, error) {
	var (
		pak packet.IPacket
		res int32
		err error
	)
	err = c.basePacketHandler.OnUpdateHandle()
	if err != nil {
		return false, err
	}
	if mode == 0 {
		pak, _, err = c.conn.Wait(c.ctx, nil)
	} else {
		pak, err = c.conn.RecvNonblock()
	}
	if err == nil && pak != nil {
		res, err = c.basePacketHandler.OnHandleHandshake(pak)
	}
	c.options.GetPacketPool().Put(pak)
	if err == common.ErrRecvChanEmpty {
		err = nil
	}
	return res == 2, err
}

func (c *Client) handle(mode int32) error {
	var (
		pak packet.IPacket
		res int32
		err error
	)

	if mode == 0 {
		pak, _, err = c.conn.Wait(c.ctx, c.sess.GetPacketChannel())
	} else {
		pak, err = c.conn.RecvNonblock()
	}
	if err == nil {
		if pak != nil { // net packet handle
			res, err = c.basePacketHandler.OnPreHandle(pak)
			if err == nil {
				if res == 0 {
					err = c.handler.OnPacket(c.sess, pak)
				}
			}
			if err == nil {
				err = c.basePacketHandler.OnPostHandle(pak)
			}
			c.options.GetPacketPool().Put(pak)
		} else { // tick handle
			c.handleTick()
			err = c.basePacketHandler.OnUpdateHandle()
		}
	}

	if mode != 0 && err == common.ErrRecvChanEmpty {
		err = nil
	}

	if err != nil {
		if common.IsNoDisconnectError(err) {
			c.handler.OnError(err)
		}
	}

	return err
}

func (c *Client) handleTick() {
	now := time.Now()
	tick := now.Sub(c.lastTime)
	c.handler.OnTick(c.sess, tick)
	c.lastTime = now
}

func (c *Client) handleErr(err error) error {
	if err != nil {
		// if no disconnect error, reset err to nil
		if common.IsNoDisconnectError(err) {
			err = nil
		} else {
			if c.packetBuilder != nil {
				c.packetBuilder.Close()
			}
			if c.handler != nil {
				c.handler.OnDisconnect(c.sess, err)
			}
			c.conn.Close()
		}
	}
	return err
}

func (c *Client) handleReconnect(lastCheck *time.Time, reconnectTimer *time.Timer) {
	since := time.Since(*lastCheck)
	if since < time.Duration(c.options.GetReconnectSeconds()) {
		if reconnectTimer == nil {
			reconnectTimer = time.NewTimer(time.Duration(c.options.GetReconnectSeconds()) - since)
		} else {
			reconnectTimer.Reset(time.Duration(c.options.GetReconnectSeconds()) - since)
		}
		<-reconnectTimer.C
	}
	*lastCheck = time.Now()
	c.reset()
	c.ConnectAsync(c.address, c.connTimeout, c.connAsyncCallback)
}

func (c *Client) getError() error {
	var err error
	if c.IsNotConnect() {
		err = ErrClientNotConnect
	} else if c.IsConnecting() {
		err = ErrClientConnecting
	} else if c.IsDisconnecting() {
		err = ErrClientDisconnecting
	} else if c.IsDisconnected() {
		err = ErrClientDisconnected
	} else if c.IsConnected() && !c.IsReady() {
		err = ErrClientNotReady
	}
	return err
}

type packetEventHandler struct {
	builder *common.PacketBuilder
	sess    *common.SessionEx
}

func (h *packetEventHandler) OnHandshakeDone(args ...any) error {
	var (
		ct     packet.CompressType
		et     packet.EncryptionType
		key    []byte
		sessId uint64
		o      bool
	)
	lenArgs := len(args)
	if lenArgs > 0 {
		ct, o = args[0].(packet.CompressType)
		if !o {
			return errors.New("gsnet: handshake complete cast compress type failed")
		}
		if lenArgs > 1 {
			et, o = args[1].(packet.EncryptionType)
			if !o {
				return errors.New("gsnet: handshake complete cast encryption type failed")
			}
			if lenArgs > 2 {
				key, o = args[2].([]byte)
				if !o {
					return errors.New("gsnet: handshake complete cast crypto key type failed")
				}
			}
		}
		h.builder.Reset(ct, et, key)
		if lenArgs > 3 {
			sessId, o = args[3].(uint64)
			if !o {
				return errors.New("gsnet: handshake complete cast session id type failed")
			}
			h.sess.SetId(sessId)
		}
	}
	return nil
}
