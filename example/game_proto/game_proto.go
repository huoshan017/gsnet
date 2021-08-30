package game_proto

const (
	MsgIdHandShakeReq        = 1
	MsgIdHandShakeAck        = 2
	MsgIdGamePlayerEnterReq  = 100
	MsgIdGamePlayerEnterResp = 101
	MsgIdGamePlayerExitReq   = 102
	MsgIdGamePlayerExitResp  = 103
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
