package kcp

import (
	"errors"
	"net"
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/log"
)

type uConn struct {
	conn     *net.UDPConn
	raddr    *net.UDPAddr
	convId   uint32
	token    int64
	state    int32
	cn       uint8
	tid      uint32
	recvList chan mBufferSlice
}

func newUConn() *uConn {
	return &uConn{}
}

func DialUDP(network, address string) (net.Conn, error) {
	addr, err := net.ResolveUDPAddr(network, address)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialUDP(network, nil, addr)
	if err != nil {
		return nil, err
	}

	var (
		c      uConn
		header frameHeader
	)

	// 三次握手
	for i := 0; i < 3; i++ {
		// send syn
		err = sendSyn(conn, &header)
		if err != nil {
			return nil, err
		}

		log.Infof("gsnet: kcp conn send syn")

		c.state = STATE_SYN_SEND

		// recv synack
		err = recvSynAck(conn, 3*time.Second, &header)
		if err != nil {
			if common.IsTimeoutError(err) {
				log.Infof("!!!!! receive synack timeout")
				continue
			}
			return nil, err
		}

		log.Infof("gsnet: kcp conn received synack, conversation(%v) token(%v)", header.convId, header.token)
		break
	}

	// send ack
	err = sendAck(conn, header.convId, header.token)
	if err != nil {
		return nil, err
	}

	c.state = STATE_ESTABLISHED
	c.conn = conn
	c.convId = header.convId
	c.token = header.token

	log.Infof("gsnet: kcp client connection established, conversation(%v) token(%v)", header.convId, header.token)

	uc := newUConn()
	*uc = c
	return uc, nil
}

func (c *uConn) Read(buf []byte) (int, error) {
	return 0, errors.New("gsnet: kcp uConn.Read call not allowed")
}

func (c *uConn) Write(buf []byte) (int, error) {
	if c.raddr != nil {
		return c.conn.WriteToUDP(buf, c.raddr)
	}
	return c.conn.Write(buf)
}

func (c *uConn) Close() error {
	c.state = STATE_CLOSED
	close(c.recvList)
	c.conn = nil
	return nil
}

func (c *uConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *uConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *uConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *uConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

func (c *uConn) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}
