package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"time"
)

func main() {

	//client()
	tlsClient()
}

func client() {
	conn, err := net.Dial("tcp", "127.0.0.1:4433")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	total := 0

	for {

		if total >= 3 {
			return
		}
		fmt.Println(conn.Write([]byte("from hello")))
		total += 1
		time.Sleep(time.Second * 5)
	}

}

func tlsClient() {
	conf := &tls.Config{
		InsecureSkipVerify: true,
	}
	conn, err := tls.Dial("tcp", "127.0.0.1:6565", conf)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	fmt.Println("以下是发送数据")
	time.Sleep(time.Second * 10)
	total := 0
	for {
		if total >= 3 {
			return
		}
		n, err := conn.Write([]byte("hello\n"))
		if err != nil {
			log.Println(n, err)
			return
		}

		fmt.Println("conn.write.n", n, err)
		//buf := make([]byte, 100)
		//n, err = conn.Read(buf)
		//if err != nil {
		//	log.Println(n, err)
		//	return
		//}
		//println(string(buf[:n]))
		total += 1
		time.Sleep(time.Second * 5)
	}
}
