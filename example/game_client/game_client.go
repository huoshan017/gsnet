package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/huoshan017/gsnet"
	"github.com/huoshan017/gsnet/example/game_proto"
)

const (
	PlayerStateNotEnter = 0
	PlayerStateEntering = 1
	PlayerStateEntered  = 2
	PlayerStateExiting  = 3
)

type IOwner interface {
	GetNet() *gsnet.MsgClient
}

type Player struct {
	owner IOwner
	state int32
}

func NewPlayer(owner IOwner) *Player {
	return &Player{
		owner: owner,
	}
}

func (p *Player) Init() {
	p.registerHandle(game_proto.MsgIdGamePlayerEnterResp, p.onEnterGame)
	p.registerHandle(game_proto.MsgIdGamePlayerExitResp, p.onExitGame)
}

func (p *Player) SendEnterGame() error {
	d, e := json.Marshal(&game_proto.GamePlayerEnterReq{})
	if e != nil {
		return e
	}
	e = p.send(game_proto.MsgIdGamePlayerEnterReq, d)
	if e != nil {
		return e
	}
	p.state = PlayerStateEntering
	fmt.Println("Player send enter message")
	return nil
}

func (p *Player) SendExitGame() error {
	d, e := json.Marshal(&game_proto.GamePlayerExitReq{})
	if e != nil {
		return e
	}
	e = p.send(game_proto.MsgIdGamePlayerExitReq, d)
	if e != nil {
		return e
	}
	p.state = PlayerStateExiting
	fmt.Println("Player send exit message")
	return nil
}

func (p *Player) send(msgid uint32, msgdata []byte) error {
	return p.owner.GetNet().Send(msgid, msgdata)
}

func (p *Player) registerHandle(msgid uint32, handle func(gsnet.ISession, []byte) error) {
	p.owner.GetNet().RegisterHandle(msgid, handle)
}

func (p *Player) onEnterGame(sess gsnet.ISession, data []byte) error {
	p.state = PlayerStateEntered
	fmt.Println("Player entered game")
	return nil
}

func (p *Player) onExitGame(sess gsnet.ISession, data []byte) error {
	p.state = PlayerStateNotEnter
	p.owner.GetNet().Close()
	fmt.Println("Player exited game")
	return nil
}

func (p *Player) onTick() {
	if p.state == PlayerStateNotEnter {
		p.SendEnterGame()
	} else if p.state == PlayerStateEntered {
		p.SendExitGame()
	}
}

type config struct {
	addr string
	tick time.Duration
}

type GameClient struct {
	self *Player
	net  *gsnet.MsgClient
}

func NewGameClient() *GameClient {
	client := &GameClient{}
	client.self = NewPlayer(client)
	return client
}

func (c *GameClient) GetNet() *gsnet.MsgClient {
	return c.net
}

func (c *GameClient) Init(conf *config) error {
	net := gsnet.NewMsgClient(gsnet.SetTickSpan(time.Millisecond))
	err := net.Connect(conf.addr)
	if err != nil {
		fmt.Println("connect ", conf.addr, " failed")
		return err
	}
	c.net = net
	c.self.Init()
	return nil
}

func (c *GameClient) OnConnect() {
}

func (c *GameClient) OnDisconnect(err error) {
	fmt.Println("GameClient::OnDisconnect err ", err)
}

func (c *GameClient) OnError(err error) {
	fmt.Println("GameClient::OnError err ", err)
}

func (c *GameClient) OnTick(tick time.Duration) {
	c.self.onTick()
}

func (c *GameClient) Run() {
	c.net.Run()
}

func (c *GameClient) Close() {
	c.net.Close()
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("args num invalid")
		return
	}
	ip_str := flag.String("ip", "", "ip set")
	flag.Parse()

	gameClient := NewGameClient()
	gameClient.Init(&config{
		addr: *ip_str,
		tick: time.Millisecond,
	})

	defer gameClient.Close()

	gameClient.Run()
}
