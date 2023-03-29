#!/bin/sh
go build -mod=mod -o ./bin/client.exe github.com/huoshan017/gsnet/example/agent/client
go build -mod=mod -o ./bin/server.exe github.com/huoshan017/gsnet/example/agent/server
go build -mod=mod -o ./bin/agent_server.exe github.com/huoshan017/gsnet/example/agent/agent_server