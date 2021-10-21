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

func (h *PlayerHandler) Init(args ...interface{}) {

}

func (h *PlayerHandler) OnConnect(s gsnet.ISession) {

}

func (h *PlayerHandler) OnDisconnect(s gsnet.ISession, err error) {

}

func (h *PlayerHandler) OnData(s gsnet.ISession, data []byte) error {
	return s.Send(data)
}

func (h *PlayerHandler) OnTick(s gsnet.ISession, tick time.Duration) {

}

func (h *PlayerHandler) OnError(err error) {

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
	h := &PlayerHandler{}
	playerService := gsnet.NewServer(h, nil, gsnet.SetErrChanLen(100))
	err := playerService.Listen(addr)
	if err != nil {
		fmt.Println("player service listen ", addr, " err: ", err)
		return
	}
	defer playerService.End()
	playerService.Start()
}
