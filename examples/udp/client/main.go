package main

import (
	"fmt"
	"github.com/ikilobyte/netman/util"
	"net"
	"os"
	"time"
)

func main() {

	//bs := []byte{1, 2, 3}
	//fmt.Println(bs[:8])
	//os.Exit(0)

	//bs := []byte{7, 0, 0, 0, 0, 0, 0, 0, 104, 101, 108, 108, 111, 32, 48}
	////
	//fmt.Println(bs[:8])
	//fmt.Println(bs[8 : 8+7])
	//fmt.Println(bs)
	//os.Exit(0)

	fmt.Println(os.Getpid())
	conn, err := net.Dial("udp", "127.0.0.1:6565")
	if err != nil {
		panic(err)
	}

	packer := util.NewDataPacker()

	total := 0
	for {

		// 封装消息
		bs, _ := packer.Pack(0, []byte(fmt.Sprintf("hello %d", total)))

		//if total >= 2 {
		//	bs = []byte("wqrqwqwr")
		//}

		// 发送消息
		n, err := conn.Write(bs)

		fmt.Println("write.n", n, "err", err, bs)
		time.Sleep(time.Second * 3)
		total += 1
	}
}
