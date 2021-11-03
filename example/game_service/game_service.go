package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/huoshan017/gsnet"
	"github.com/huoshan017/gsnet/example/game_proto"

	cmap "github.com/orcaman/concurrent-map"
)

var ErrKickDuplicatePlayer = errors.New("game service example: kick duplicate player")

type Player struct {
	id      int
	account string
	token   string
	sess    gsnet.ISession
}

func NewPlayer() *Player {
	return &Player{}
}

func (p *Player) GetId() int {
	return p.id
}

func (p *Player) GetAccount() string {
	return p.account
}

func (p *Player) GetToken() string {
	return p.token
}

type PlayerManager struct {
	id2player cmap.ConcurrentMap
}

func NewPlayerManager() *PlayerManager {
	return &PlayerManager{
		id2player: cmap.New(),
	}
}

var playerMgr = NewPlayerManager()

func (pm *PlayerManager) Add(p *Player) {
	pm.id2player.Set(strconv.Itoa(p.id), p)
}

func (pm *PlayerManager) Remove(pid int) {
	pm.id2player.Remove(strconv.Itoa(pid))
}

func (pm *PlayerManager) Get(pid int) *Player {
	p, o := pm.id2player.Get(strconv.Itoa(pid))
	if !o {
		return nil
	}
	return p.(*Player)
}

type config struct {
	addr string
}

type DefaultMsgHandler struct {
	gsnet.MsgHandler
	msgProto gsnet.IMsgProto
}

func (h *DefaultMsgHandler) Init(args ...interface{}) {
	h.msgProto = &gsnet.DefaultMsgProto{}
	h.MsgHandler = *gsnet.NewMsgHandler(h.msgProto)
	h.RegisterHandle(game_proto.MsgIdGamePlayerEnterReq, h.onPlayerEnterGame)
	h.RegisterHandle(game_proto.MsgIdGamePlayerExitReq, h.onPlayerExitGame)
	h.RegisterHandle(game_proto.MsgIdHandShakeReq, h.onHandShake)
}

func (h *DefaultMsgHandler) OnConnect(sess gsnet.ISession) {
	log.Printf("session %v connected", sess.GetId())
}

func (h *DefaultMsgHandler) OnDisconnect(sess gsnet.ISession, err error) {
	log.Printf("session %v disconnected", sess.GetId())
}

func (h *DefaultMsgHandler) OnTick(sess gsnet.ISession, tick time.Duration) {
}

func (h *DefaultMsgHandler) OnError(err error) {
	log.Printf("err %v", err)
}

func (h *DefaultMsgHandler) onHandShake(sess gsnet.ISession, data []byte) error {
	return nil
}

func (h *DefaultMsgHandler) onPlayerEnterGame(sess gsnet.ISession, data []byte) error {
	var req game_proto.GamePlayerEnterReq
	err := json.Unmarshal(data, &req)
	var resp game_proto.GamePlayerEnterResp
	if err != nil {
		resp.Result = -1
		d, e := json.Marshal(&resp)
		if e != nil {
			return e
		}
		return h.Send(sess, game_proto.MsgIdGamePlayerEnterResp, d)
	}

	var p *Player
	// 先判断session中有没保存的数据
	pd := sess.GetData("player")
	if pd != nil {
		// 重复进入
		resp.Result = -2
		d, e := json.Marshal(&resp)
		if e != nil {
			return e
		}
		e = h.Send(sess, game_proto.MsgIdGamePlayerEnterResp, d)
		if e != nil {
			return e
		}
		var o bool
		p, o = pd.(*Player)
		if !o {
			return errors.New("game_service: type cast to *Player failed")
		}
		fmt.Println("error: same player, kick it")
		// 断开之前的连接
		p.sess.Close()
	}
	p = NewPlayer()
	p.account = req.Account
	p.token = req.Token
	p.sess = sess
	playerMgr.Add(p)
	sess.SetData("player", pd)
	dd, e := json.Marshal(&game_proto.GamePlayerEnterResp{})
	if e != nil {
		return e
	}
	h.Send(sess, game_proto.MsgIdGamePlayerEnterResp, dd)
	fmt.Println("Player ", p.account, " entered game")
	return nil
}

func (h *DefaultMsgHandler) onPlayerExitGame(sess gsnet.ISession, data []byte) error {
	d := sess.GetData("player")
	if d == nil {
		return errors.New("game_service: no invalid session")
	}
	p, o := d.(*Player)
	if !o {
		return errors.New("game_service: type cast to Player failed")
	}
	var resp game_proto.GamePlayerExitResp
	dd, e := json.Marshal(&resp)
	if e != nil {
		return e
	}
	h.Send(sess, game_proto.MsgIdGamePlayerExitResp, dd)
	fmt.Println("player ", p.account, " exit game")
	return nil
}

type GameService struct {
	net *gsnet.Server
}

func NewGameService() *GameService {
	return &GameService{}
}

func (s *GameService) GetNet() *gsnet.Server {
	return s.net
}

func (s *GameService) Init(conf *config) bool {
	h := &DefaultMsgHandler{}
	net := gsnet.NewServer(h, nil)
	err := net.Listen(conf.addr)
	if err != nil {
		fmt.Println("game service listen addr ", conf.addr, " err: ", err)
		return false
	}
	s.net = net
	return true
}

func (s *GameService) Start() {
	s.net.Start()
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("args num invalid")
		return
	}
	ip_str := flag.String("ip", "", "ip set")
	flag.Parse()

	gameService := NewGameService()
	if !gameService.Init(&config{addr: *ip_str}) {
		return
	}

	gameService.Start()
}
