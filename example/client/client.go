package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/huoshan017/gsnet"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("args num invalid")
		return
	}

	ip_str := flag.String("i", "", "ip set")
	if ip_str == nil {
		fmt.Println("err: ip not set")
		return
	}
	port_str := flag.String("p", "", "port set")
	if port_str == nil {
		fmt.Println("err: port not set")
		return
	}
	player_num := flag.Int("n", 1, "player num set")
	send_num := flag.Int("s", 1, "send num set")
	flag.Parse()

	addr := *ip_str + ":" + *port_str
	data := []byte(test_string)
	rand.Seed(time.Now().Unix())
	for n := 1; n <= *player_num; n++ {
		go func(no int) {
			conn := gsnet.NewConnector(&gsnet.ConnOptions{DataProto: &gsnet.DefaultDataProto{}})
			err := conn.Connect(addr)
			if err != nil {
				fmt.Println("connector Connect err: ", err)
				return
			}
			fmt.Println("connected: ", addr)
			defer conn.Close()

			comp_num := int32(0)
			for i := 0; i < *send_num; i++ {
				s := rand.Intn(len(data))
				e := rand.Intn(len(data))
				if e == s {
					e = s + rand.Intn(10)
				} else if e < s {
					e, s = s, e
				}
				fmt.Println("connector ", no, " ready to send data length is ", len(data[s:e]), ", compared ", comp_num)
				if err = conn.Send(data[s:e]); err != nil {
					fmt.Println("connector send data err: ", err)
					return
				}
				var recv_data []byte
				recv_data, err = conn.Recv()
				if err != nil {
					fmt.Println("connector ", no, " recv data err: ", err)
					return
				}
				if string(recv_data) != string(data[s:e]) {
					str := fmt.Sprintf("invalid received string len %v != len %v, connector %v compared num %v",
						len(string(recv_data)), len(string(data[s:e])), no, comp_num)
					panic(str)
				}

				comp_num += 1
				time.Sleep(time.Microsecond)
			}
		}(n)
	}

	for {
		time.Sleep(time.Second)
	}
}
