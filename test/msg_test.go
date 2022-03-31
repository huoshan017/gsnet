package test

import (
	"fmt"
	"testing"
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/msg"

	"github.com/huoshan017/gsnet/test/tproto"
)

const (
	MsgIdPing = msg.MsgIdType(1)
	MsgIdPong = msg.MsgIdType(2)
	sendCount = 100
)

var idMsgMapper *msg.IdMsgMapper = msg.CreateIdMsgMapper()

var (
	ch = make(chan struct{})
)

func init() {
	//idMsgMapper.AddMap()
}

func newPBMsgClient(t *testing.T) (*msg.MsgClient, error) {
	c := msg.NewPBMsgClient(idMsgMapper, common.WithTickSpan(10*time.Millisecond))
	err := c.Connect(testAddress)
	if err != nil {
		return nil, fmt.Errorf("TestPBMsgClient connect address %v err %v", testAddress, err)
	}

	c.SetConnectHandle(func(sess common.ISession) {
		t.Logf("connected")
	})

	c.SetDisconnectHandle(func(sess common.ISession, err error) {
		t.Logf("disconnected, err %v", err)
	})

	var n int
	c.SetTickHandle(func(sess common.ISession, tick time.Duration) {
		if n < sendCount {
			var ping tproto.MsgPing
			ping.Content = "pingpingping"
			c.Send(MsgIdPing, &ping)
			n += 1
		} else {
			ch <- struct{}{}
		}
	})

	c.SetErrorHandle(func(err error) {
		t.Logf("get error: %v", err)
	})

	var rn int
	c.RegisterMsgHandle(MsgIdPong, func(sess common.ISession, msg interface{}) error {
		t.Logf("received Pong message %v", rn)
		rn += 1
		return nil
	})

	return c, nil
}

func newPBMsgServer(t *testing.T) (*msg.MsgServer, error) {
	s := msg.NewPBMsgServer(idMsgMapper)
	err := s.Listen(testAddress)
	if err != nil {
		return nil, err
	}
	s.SetConnectHandle(func(sess common.ISession) {
		t.Logf("session %v connected", sess.GetId())
	})

	s.SetDisconnectHandle(func(sess common.ISession, err error) {
		t.Logf("session %v disconnected", sess.GetId())
	})

	s.SetTickHandle(func(sess common.ISession, tick time.Duration) {

	})

	s.SetErrorHandle(func(err error) {
		t.Logf("session err: %v", err)
	})

	s.RegisterMsgHandle(MsgIdPing, func(sess common.ISession, msg interface{}) error {
		m, o := msg.(*tproto.MsgPing)
		if !o {
			t.Errorf("server receive message must Ping")
		}
		t.Logf("received session %v message %v", sess.GetId(), m.Content)
		var rm tproto.MsgPong
		rm.Content = "pongpongpong"
		return s.Send(sess, MsgIdPong, &rm)
	})
	s.Start()
	return s, nil
}

func TestPBMsgClient(t *testing.T) {
	s, err := newPBMsgServer(t)
	if err != nil {
		t.Errorf("%v", err)
		return
	}
	err = s.Listen(testAddress)
	if err != nil {
		t.Errorf("%v", err)
		return
	}
	defer s.End()
	go s.Start()

	c, err := newPBMsgClient(t)
	if err != nil {
		t.Errorf("%v", err)
		return
	}
	go func() {
		<-ch
		c.Close()
	}()
	c.Run()
}

func TestPBMsgServer(t *testing.T) {

}
