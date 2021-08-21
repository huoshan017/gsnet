package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/huoshan017/gsnet"
	"github.com/huoshan017/gsnet/example/game_proto"
)

type Player struct {
	id    int
	acc   string
	token string
	c     *gsnet.MsgClient
}

func NewPlayer() *Player {
	return &Player{}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("args num invalid")
		return
	}
	ip_str := flag.String("ip", "", "ip set")
	flag.Parse()

	c := gsnet.NewMsgClient(&gsnet.DefaultClientCallback{})
	c.RegisterHandle(game_proto.MsgIdGamePlayerEnterResp, func(s *gsnet.Session, data []byte) error {
		return nil
	})
	c.RegisterHandle(game_proto.MsgIdGamePlayerExitResp, func(s *gsnet.Session, data []byte) error {
		return nil
	})
	err := c.Connect(*ip_str)
	if err != nil {
		fmt.Println("connect ", *ip_str, " failed")
		return
	}

	c.Run()
}
