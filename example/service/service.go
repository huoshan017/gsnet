package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/huoshan017/gsnet"
)

type PlayerHandler struct {
}

func (h *PlayerHandler) OnData(s *gsnet.Session, data []byte) error {
	return s.Send(data)
}

type PlayerCallback struct {
}

func (p *PlayerCallback) OnConnect(sid uint64) {
	fmt.Println("PlayerCallback::OnConnect session", sid, " connected")
}

func (p *PlayerCallback) OnDisconnect(sid uint64, err error) {
	fmt.Println("PlayerCallback::OnDisconnect session", sid, " disconnected err: ", err)
}

func (p *PlayerCallback) OnError(err error) {
	fmt.Println("PlayerCallback::OnError err: ", err)
}

func (p *PlayerCallback) OnTick(tick time.Duration) {
	fmt.Println("PlayerCallback::OnTick")
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("args num invalid")
		return
	}
	ip_str := flag.String("i", "", "ip set")
	port_str := flag.String("p", "", "port set")
	flag.Parse()

	addr := *ip_str + ":" + *port_str
	cb := &PlayerCallback{}
	h := &PlayerHandler{}
	playerService := gsnet.NewService(cb, h, gsnet.SetErrChanLen(100))
	err := playerService.Listen(addr)
	if err != nil {
		fmt.Println("player service listen ", addr, " err: ", err)
		return
	}
	defer playerService.End()
	playerService.Start()
}
