#!/bin/bash

go build -mod=mod -o ./bin/client github.com/huoshan017/gsnet/example/client
go build -mod=mod -o ./bin/server github.com/huoshan017/gsnet/example/server
go build -mod=mod -o ./bin/service github.com/huoshan017/gsnet/example/service
go build -mod=mod -o ./bin/game_service github.com/huoshan017/gsnet/example/game_service
go build -mod=mod -o ./bin/game_client github.com/huoshan017/gsnet/example/game_client