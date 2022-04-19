package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/packet"
	"github.com/huoshan017/gsnet/server"
)

type PlayerHandler struct {
}

func (h *PlayerHandler) OnConnect(s common.ISession) {

}

func (h *PlayerHandler) OnDisconnect(s common.ISession, err error) {

}

func (h *PlayerHandler) OnPacket(s common.ISession, p packet.IPacket) error {
	return s.Send(p.Data(), true)
}

func (h *PlayerHandler) OnTick(s common.ISession, tick time.Duration) {

}

func (h *PlayerHandler) OnError(err error) {

}

func NewPlayerHandler(args ...any) common.ISessionEventHandler {
	return &PlayerHandler{}
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
	playerService := server.NewServer(NewPlayerHandler)
	err := playerService.Listen(addr)
	if err != nil {
		fmt.Println("player service listen ", addr, " err: ", err)
		return
	}
	defer playerService.End()
	playerService.Start()
}
