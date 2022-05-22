package main

import (
	"net/http"

	"github.com/huoshan017/gsnet/example/framework/common"
	"github.com/huoshan017/gsnet/framework/frontend"

	_ "net/http/pprof"
)

func main() {
	go func() {
		http.ListenAndServe("0.0.0.0:6060", nil)
	}()
	gate := frontend.NewGate(frontend.NewGateOptions(common.BackendAddress, frontend.RouteTypeRandom))
	gate.ListenAndServe(common.GateAddress)
}
