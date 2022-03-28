package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/server"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("args num invalid")
		return
	}
	ip_str := flag.String("i", "", "ip set")
	port_str := flag.String("p", "", "port set")
	flag.Parse()

	addr := *ip_str + ":" + *port_str
	acceptor := server.NewAcceptor(server.ServerOptions{})
	err := acceptor.Listen(addr)
	if err != nil {
		fmt.Println("acceptor listen addr: ", addr, " err: ", err)
		return
	}
	defer acceptor.Close()

	fmt.Println("listening: ", addr)

	go acceptor.Serve()

	c := 0
	var o bool = true
	var conn common.IConn
	for o {
		select {
		case conn, o = <-acceptor.GetNewConnChan():
			if !o {
				continue
			}
			conn.Run()
			c += 1
			go func(no int, conn common.IConn) {
				fmt.Println("connection ", no)
				for {
					packet, e := conn.Recv()
					if e != nil {
						conn.Close()
						fmt.Println("conn ", no, " recv err: ", e)
						break
					}
					e = conn.Send(*packet.Data(), true)
					if e != nil {
						conn.Close()
						fmt.Println("conn ", no, " send err: ", e)
						break
					}
					time.Sleep(time.Microsecond)
				}
			}(c, conn)
		default:
		}
	}

	for {
		time.Sleep(time.Second)
	}
}
