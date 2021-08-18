package gsnet

import (
	"net"
)

type Connector struct {
	*Conn
	options ConnOptions
}

func NewConnector(options *ConnOptions) *Connector {
	c := &Connector{
		options: *options,
	}
	return c
}

func (c *Connector) Connect(address string) error {
	var conn net.Conn
	var err error
	conn, err = net.Dial("tcp", address)
	if err != nil {
		return err
	}
	c.Conn = NewConn(conn, &c.options)
	c.Run()
	return nil
}

func (c *Connector) Close() error {
	return c.conn.Close()
}
