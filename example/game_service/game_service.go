package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/example/game_proto"
	"github.com/huoshan017/gsnet/msg"

	cmap "github.com/orcaman/concurrent-map"
)

var ErrKickDuplicatePlayer = errors.New("game service example: kick duplicate player")

type Player struct {
	id      int
	account string
	token   string
	sess    *msg.MsgSession
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

type SessionHandler struct{}

func (h *SessionHandler) OnConnect(sess common.ISession) {
	log.Printf("session %v connected", sess.GetId())
}

func (h *SessionHandler) OnDisconnect(sess common.ISession, err error) {
	log.Printf("session %v disconnected", sess.GetId())
}

func (h *SessionHandler) OnTick(sess common.ISession, tick time.Duration) {
}

func (h *SessionHandler) OnError(err error) {
	log.Printf("err %v", err)
}

func (h *SessionHandler) onPlayerEnterGame(sess *msg.MsgSession, msg interface{}) error {
	req, o := msg.(*game_proto.GamePlayerEnterReq)
	if !o {
		panic("onPlayerEnterGame must receive game_proto.GamePlayerEnterReq")
	}
	var resp game_proto.GamePlayerEnterResp
	var p *Player
	// 先判断session中有没保存的数据
	pd := sess.GetData("player")
	if pd != nil {
		// 重复进入
		resp.Result = -2
		e := sess.SendMsg(game_proto.MsgIdGamePlayerEnterResp, &resp)
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
	sess.SendMsg(game_proto.MsgIdGamePlayerEnterResp, &resp)
	fmt.Println("Player ", p.account, " entered game")
	return nil
}

func (h *SessionHandler) onPlayerExitGame(sess *msg.MsgSession, msg interface{}) error {
	d := sess.GetData("player")
	if d == nil {
		return errors.New("game_service: no invalid session")
	}
	p, o := d.(*Player)
	if !o {
		return errors.New("game_service: type cast to Player failed")
	}
	var resp game_proto.GamePlayerExitResp
	sess.SendMsg(game_proto.MsgIdGamePlayerExitResp, &resp)
	fmt.Println("player ", p.account, " exit game")
	return nil
}

type GameService struct {
	serv        *msg.MsgServer
	sessHandles *SessionHandler
}

func NewGameService() *GameService {
	return &GameService{}
}

func (s *GameService) GetNet() *msg.MsgServer {
	return s.serv
}

func (s *GameService) Init(conf *config) bool {
	serv := msg.NewGobMsgServer(msg.CreateIdMsgMapper())
	err := serv.Listen(conf.addr)
	if err != nil {
		fmt.Println("game service listen addr ", conf.addr, " err: ", err)
		return false
	}
	s.serv = serv
	s.setHandles()
	return true
}

func (s *GameService) setHandles() {
	sessionHandler := &SessionHandler{}
	s.serv.SetSessionConnectedHandle(sessionHandler.OnConnect)
	s.serv.SetSessionDisconnectedHandle(sessionHandler.OnDisconnect)
	s.serv.SetSessionTickHandle(sessionHandler.OnTick)
	s.serv.SetSessionErrorHandle(sessionHandler.OnError)
	s.serv.SetMsgSessionHandle(game_proto.MsgIdGamePlayerEnterReq, sessionHandler.onPlayerEnterGame)
	s.serv.SetMsgSessionHandle(game_proto.MsgIdGamePlayerExitReq, sessionHandler.onPlayerExitGame)
	s.sessHandles = sessionHandler
}

func (s *GameService) Start() {
	s.serv.Start()
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
