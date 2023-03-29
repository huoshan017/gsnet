#!/bin/sh
go build -mod=mod -o ./bin/client.exe github.com/huoshan017/gsnet/example/framework/client
go build -mod=mod -o ./bin/frontend.exe github.com/huoshan017/gsnet/example/framework/frontend
go build -mod=mod -o ./bin/backend.exe github.com/huoshan017/gsnet/example/framework/backend