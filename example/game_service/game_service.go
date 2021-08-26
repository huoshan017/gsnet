package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
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
	sess    *gsnet.Session
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

func (p *Player) GetSessionId() uint64 {
	return p.sess.GetId()
}

func (p *Player) Exit() {
	p.sess.Close()
}

type SessionMgr struct {
	sid2player cmap.ConcurrentMap
}

func NewSessionMgr() *SessionMgr {
	return &SessionMgr{
		sid2player: cmap.New(),
	}
}

var sessMgr = NewSessionMgr()

func (sm *SessionMgr) GetPlayer(sid uint64) *Player {
	p, o := sm.sid2player.Get(strconv.FormatUint(sid, 10))
	if !o {
		return nil
	}
	return p.(*Player)
}

func (sm *SessionMgr) AddPlayer(sid uint64, p *Player) {
	sm.sid2player.Set(strconv.FormatUint(sid, 10), p)
}

func (sm *SessionMgr) RemovePlayer(sid uint64) {
	sm.sid2player.Get(strconv.FormatUint(sid, 10))
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

type GameService struct {
	net *gsnet.MsgService
}

func NewGameService() *GameService {
	return &GameService{}
}

func (s *GameService) GetNet() *gsnet.MsgService {
	return s.net
}

func (s *GameService) Init(conf *config) bool {
	net := gsnet.NewMsgService(s)
	err := net.Listen(conf.addr)
	if err != nil {
		fmt.Println("game service listen addr ", conf.addr, " err: ", err)
		return false
	}
	s.net = net
	s.registerHandle(game_proto.MsgIdGamePlayerEnterReq, s.onPlayerEnterGame)
	s.registerHandle(game_proto.MsgIdGamePlayerExitReq, s.onPlayerExitGame)
	return true
}

func (s *GameService) Start() {
	s.net.Start()
}

func (s *GameService) OnConnect(sessId uint64) {
	fmt.Println("session ", sessId, " connected")
}

func (s *GameService) OnDisconnect(sessId uint64, err error) {
	fmt.Println("session ", sessId, " disconnect, err ", err)
}

func (s *GameService) OnError(err error) {
	fmt.Println("error: ", err)
}

func (s *GameService) OnTick(tick time.Duration) {
	fmt.Println("onTick")
}

func (s *GameService) registerHandle(msgid uint32, handle func(*gsnet.Session, []byte) error) {
	s.net.RegisterHandle(msgid, handle)
}

func (s *GameService) onPlayerEnterGame(sess *gsnet.Session, data []byte) error {
	var req game_proto.GamePlayerEnterReq
	err := json.Unmarshal(data, &req)
	var resp game_proto.GamePlayerEnterResp
	if err != nil {
		resp.Result = -1
		d, e := json.Marshal(&resp)
		if e != nil {
			return e
		}
		return s.net.Send(sess, game_proto.MsgIdGamePlayerEnterResp, d)
	}
	p := sessMgr.GetPlayer(sess.GetId())
	if p == nil {
		p = NewPlayer()
		p.sess = sess
		sessMgr.AddPlayer(sess.GetId(), p)
	} else {
		resp.Result = -2
		d, e := json.Marshal(&resp)
		if e != nil {
			return e
		}
		e = s.net.Send(sess, game_proto.MsgIdGamePlayerEnterResp, d)
		if e != nil {
			return e
		}
		fmt.Println("error: same player, kick it")
		return ErrKickDuplicatePlayer
	}
	d, e := json.Marshal(&game_proto.GamePlayerEnterResp{})
	if e != nil {
		return e
	}
	s.net.Send(sess, game_proto.MsgIdGamePlayerEnterResp, d)
	fmt.Println("player ", p.account, " entered game")
	return nil
}

func (s *GameService) onPlayerExitGame(sess *gsnet.Session, data []byte) error {
	p := sessMgr.GetPlayer(sess.GetId())
	if p == nil {
		fmt.Println("error: not found player with session id ", sess.GetId())
		return nil
	}
	sessMgr.RemovePlayer(sess.GetId())
	var resp game_proto.GamePlayerExitResp
	d, e := json.Marshal(&resp)
	if e != nil {
		return e
	}
	s.net.Send(sess, game_proto.MsgIdGamePlayerExitResp, d)
	fmt.Println("player ", p.account, " exit game")
	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("args num invalid")
		return
	}
	ip_str := flag.String("ip", "", "ip set")
	flag.Parse()

	// 错误注册
	//netlib.RegisterNoDisconnectError(ErrKickDuplicatePlayer)

	gameService := NewGameService()
	if !gameService.Init(&config{addr: *ip_str}) {
		return
	}

	gameService.Start()
}
