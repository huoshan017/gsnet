package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("args num invalid")
		return
	}
	//ip_str := flag.String("ip", "", "ip set")
	flag.Parse()
}
