package game_proto

const (
	MsgIdGamePlayerEnterReq  = 1
	MsgIdGamePlayerEnterResp = 2
	MsgIdGamePlayerExitReq   = 3
	MsgIdGamePlayerExitResp  = 4
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
