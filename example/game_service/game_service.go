package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"

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

func main() {
	if len(os.Args) < 2 {
		fmt.Println("args num invalid")
		return
	}
	ip_str := flag.String("ip", "", "ip set")
	flag.Parse()

	// 错误注册
	//netlib.RegisterNoDisconnectError(ErrKickDuplicatePlayer)

	gameService := gsnet.NewMsgService(&gsnet.DefaultServiceCallback{})
	err := gameService.Listen(*ip_str)
	if err != nil {
		fmt.Println("game service listen err: ", err)
		return
	}

	// 注册进入游戏的处理函数
	gameService.RegisterHandle(game_proto.MsgIdGamePlayerEnterReq, func(s *gsnet.Session, b []byte) error {
		var req game_proto.GamePlayerEnterReq
		err := json.Unmarshal(b, &req)
		var resp game_proto.GamePlayerEnterResp
		if err != nil {
			resp.Result = -1
			d, e := json.Marshal(&resp)
			if e != nil {
				return e
			}
			return gameService.Send(s, game_proto.MsgIdGamePlayerEnterResp, d)
		}
		p := sessMgr.GetPlayer(s.GetId())
		if p == nil {
			p = NewPlayer()
			p.sess = s
			sessMgr.AddPlayer(s.GetId(), p)
		} else {
			resp.Result = -2
			d, e := json.Marshal(&resp)
			if e != nil {
				return e
			}
			e = gameService.Send(s, game_proto.MsgIdGamePlayerEnterResp, d)
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
		gameService.Send(s, game_proto.MsgIdGamePlayerEnterResp, d)
		fmt.Println("player ", p.account, " entered game")
		return nil
	})

	// 注册退出游戏的处理函数
	gameService.RegisterHandle(game_proto.MsgIdGamePlayerExitReq, func(s *gsnet.Session, b []byte) error {
		p := sessMgr.GetPlayer(s.GetId())
		if p == nil {
			fmt.Println("error: not found player with session id ", s.GetId())
			return nil
		}
		p.Exit()
		sessMgr.RemovePlayer(s.GetId())
		var resp game_proto.GamePlayerExitResp
		d, e := json.Marshal(&resp)
		if e != nil {
			return e
		}
		gameService.Send(s, game_proto.MsgIdGamePlayerExitResp, d)
		fmt.Println("player ", p.account, " exit game")
		return nil
	})

	gameService.Start()
}
