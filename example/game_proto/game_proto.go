package game_proto

const (
	MsgIdGamePlayerEnterReq  = 1
	MsgIdGamePlayerEnterResp = 2
	MsgIdGamePlayerExitReq   = 3
	MsgIdGamePlayerExitResp  = 4
)

type GamePlayerEnterReq struct {
	Account string
	Token   string
}

type GamePlayerEnterResp struct {
	Result int
}

type GamePlayerExitReq struct {
}

type GamePlayerExitResp struct {
}
