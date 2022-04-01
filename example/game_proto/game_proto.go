package game_proto

import "github.com/huoshan017/gsnet/msg"

const (
	MsgIdHandShakeReq        = msg.MsgIdType(1)
	MsgIdHandShakeAck        = msg.MsgIdType(2)
	MsgIdGamePlayerEnterReq  = msg.MsgIdType(100)
	MsgIdGamePlayerEnterResp = msg.MsgIdType(101)
	MsgIdGamePlayerExitReq   = msg.MsgIdType(102)
	MsgIdGamePlayerExitResp  = msg.MsgIdType(103)
)

type HandShakeReq struct {
}

type HandShakeAck struct {
}

type GamePlayerEnterReq struct {
	Account string
	Token   string
}

type GamePlayerEnterResp struct {
	Result int
}

type GamePlayerEnterCompleteNotify struct {
}

type GamePlayerExitReq struct {
}

type GamePlayerExitResp struct {
}

type GamePlayerBaseInfoReq struct {
}

type GamePlayerBaseInfoResp struct {
	Id    uint32
	Nick  string
	Level int32
	Pos   []int32
}

type GamePlayerItemsInfoReq struct {
}

type PlayerItemInfo struct {
	Id     int32
	InstId int32
	Count  int32
}

type GamePlayerItemsInfoResp struct {
	ItemList []*PlayerItemInfo
}
