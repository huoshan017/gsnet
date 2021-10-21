set GOPROXY="https://goproxy.io,direct"
go build -mod=mod -o ./bin/client.exe github.com/huoshan017/gsnet/example/client
go build -mod=mod -o ./bin/server.exe github.com/huoshan017/gsnet/example/server
go build -mod=mod -o ./bin/service.exe github.com/huoshan017/gsnet/example/service
go build -gcflags "-N -l" -mod=mod -o ./bin/game_service.exe github.com/huoshan017/gsnet/example/game_service
go build -mod=mod -o ./bin/game_client.exe github.com/huoshan017/gsnet/example/game_client