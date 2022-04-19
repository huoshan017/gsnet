package client

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/packet"
)

var (
	ErrClientRunUpdateMode   = errors.New("gsnet: client run update mode")
	ErrClientRunMainLoopMode = errors.New("gsnet: client run main loop mode")
)

// 数据客户端
type Client struct {
	connector         *Connector
	conn              common.IConn
	sess              common.ISession
	handler           common.ISessionEventHandler
	basePacketHandler common.IBasePacketHandler
	resend            *common.ResendData
	options           ClientOptions
	lastTime          time.Time
	ctx               context.Context
	cancel            context.CancelFunc
}

func NewClient(handler common.ISessionEventHandler, options ...common.Option) *Client {
	c := &Client{
		handler: handler,
	}
	c.ctx, c.cancel = context.WithCancel(context.Background())
	for _, option := range options {
		option(&c.options.Options)
	}
	if c.options.GetPacketPool() == nil {
		c.options.SetPacketPool(packet.GetDefaultPacketPool())
	}
	return c
}

func (c *Client) Connect(addr string) error {
	connector := c.newConnector()
	conn, err := connector.Connect(addr)
	c.doConnectResult(conn, err)
	return err
}

func (c *Client) ConnectWithTimeout(addr string, timeout time.Duration) error {
	connector := c.newConnector()
	conn, err := connector.ConnectWithTimeout(addr, timeout)
	c.doConnectResult(conn, err)
	return err
}

func (c *Client) ConnectAsync(addr string, timeout time.Duration, callback func(error)) {
	connector := c.newConnector()
	connector.ConnectAsync(addr, timeout, func(err error) {
		callback(err)
		c.doConnectResult(connector.GetConn(), err)
	})
}

func (c *Client) newConnector() *Connector {
	c.connector = NewConnector(&c.options.Options)
	return c.connector
}

func (c *Client) doConnectResult(con net.Conn, err error) {
	var (
		packetBuilder *common.DefaultPacketBuilder
		resend        *common.ResendData
	)

	switch c.options.GetConnDataType() {
	case 1:
		c.conn = common.NewConn(con, c.options.Options)
	default:
		packetBuilder = common.NewDefaultPacketBuilder(&c.options.Options)
		resendConfig := c.options.GetResendConfig()
		if resendConfig != nil {
			resend = common.NewResendData(resendConfig)
		}
		if resend != nil {
			c.conn = common.NewConn2UseResend(con, packetBuilder, resend, &c.options.Options)
		} else {
			c.conn = common.NewConn2(con, packetBuilder, &c.options.Options)
		}
	}

	c.sess = common.NewSessionNoId(c.conn)

	// 创建包事件处理器
	var pakEvtHandler common.IPacketEventHandler
	if packetBuilder != nil {
		pakEvtHandler = &packetEventHandler{handler: packetBuilder}
	}

	// 重传事件处理器
	var resendEventHandler common.IResendEventHandler
	if c.resend != nil {
		resendEventHandler = resend
	}

	// 基础包处理器
	c.basePacketHandler = common.NewDefaultBasePacketHandler4Client(c.conn, pakEvtHandler, resendEventHandler, &c.options.Options)

	// 连接跑起来
	c.conn.Run()

	// update模式下先把握手处理掉
	if c.options.GetRunMode() == RunModeOnlyUpdate {
		for {
			res, err := c.handleHandshake(0)
			if err != nil || res {
				break
			}
		}
		if err == nil {
			c.handler.OnConnect(c.sess)
		}
	}
}

func (c *Client) Send(data []byte, copyData bool) error {
	return c.sess.Send(data, copyData)
}

func (c *Client) Update() error {
	if c.options.GetRunMode() != RunModeOnlyUpdate {
		return ErrClientRunUpdateMode
	}

	// 连接状态
	if c.sess == nil {
		c.connector.WaitResult(0)
		return nil
	}

	err := c.handle(1)
	return c.handleErr(err)
}

func (c *Client) Run() {
	if c.options.GetRunMode() != RunModeAsMainLoop {
		c.handler.OnError(ErrClientRunMainLoopMode)
		return
	}

	var (
		res bool
		err error
	)

	for {
		res, err = c.handleHandshake(0)
		if err != nil || res {
			break
		}
	}

	if err == nil {
		c.handler.OnConnect(c.sess)
	}

	for err == nil {
		err = c.handle(0)
	}

	c.handleErr(err)
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
}

func (c *Client) CloseWait(secs int) {
	c.conn.CloseWait(secs)
}

func (c *Client) GetSession() common.ISession {
	return c.sess
}

func (c *Client) IsConnecting() bool {
	if c.conn == nil {
		return false
	}
	return c.connector.IsConnecting()
}

func (c *Client) IsConnected() bool {
	if c.conn == nil {
		return false
	}
	return c.connector.IsConnected()
}

func (c *Client) IsDisconnected() bool {
	return c.conn == nil || c.connector.IsDisconnected()
}

func (c *Client) IsDisconnecting() bool {
	if c.conn == nil {
		return false
	}
	return c.connector.IsDisconnecting()
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
		pak, err = c.conn.Wait(c.ctx)
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
		pak, err = c.conn.Wait(c.ctx)
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
			if c.handler != nil {
				c.handler.OnDisconnect(c.sess, err)
			}
			c.conn.Close()
		}
	}
	return err
}

type packetEventHandler struct {
	handler *common.DefaultPacketBuilder
}

func (h *packetEventHandler) OnHandshakeDone(args ...any) error {
	var (
		ct  packet.CompressType
		et  packet.EncryptionType
		key []byte
		o   bool
	)
	ct, o = args[0].(packet.CompressType)
	if !o {
		return errors.New("gsnet: handshake complete cast compress type failed")
	}
	et, o = args[1].(packet.EncryptionType)
	if !o {
		return errors.New("gsnet: handshake complete cast encryption type failed")
	}
	key, o = args[2].([]byte)
	if !o {
		return errors.New("gsnet: handshake complete cast crypto key type failed")
	}
	h.handler.Reset(ct, et, key)
	return nil
}
