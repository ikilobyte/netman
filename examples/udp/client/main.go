package main

import (
	"fmt"
	"net"
	"os"
	"time"
)

func main() {

	fmt.Println(os.Getpid())
	//time.Sleep(time.Second * 30)
	//addr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:6565")
	conn, err := net.Dial("udp", "127.0.0.1:6565")
	if err != nil {
		panic(err)
	}

	fmt.Println("conn", conn)
	for {
		n, err := conn.Write([]byte("hello world"))
		fmt.Println("write.n", n, "err", err)
		//bs := make([]byte, 1024)
		//n, err = conn.Read(bs)
		//fmt.Println("read", n, err, bs[:n])
		time.Sleep(time.Second * 1)
	}

	//time.Sleep(time.Second * 3)
	//data := make([]byte, 1024)
	//n, err = conn.Read(data)
	if err != nil {
		panic(err)
	}
	//fmt.Println(data[:n], n)
	time.Sleep(time.Hour)
}
