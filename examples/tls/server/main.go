package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

func main() {

	fmt.Println("os.Getpid()", os.Getpid())
	server()
}

func server() {

	listener, err := net.Listen("tcp", ":4433")
	if err != nil {
		panic(err)
	}

	defer listener.Close()

	cert, err := tls.LoadX509KeyPair("cert.pem", "key.pem")
	if err != nil {
		log.Println(err)
		return
	}
	config := &tls.Config{Certificates: []tls.Certificate{cert}}

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}

		fmt.Println(conn)

		// 转成tls connect
		c := tls.Server(conn, config)

		go func() {
			for {
				dataBuff := make([]byte, 65535)
				n, err := c.Read(dataBuff)
				fmt.Println("err", err)
				if n == 0 {
					_ = c.Close()
					fmt.Println("conn is closed")
					return
				}

				if err != nil {
					fmt.Println(err)
					continue
				}
				fmt.Println(n, dataBuff[:n], string(dataBuff[:n]))
			}
		}()
	}
}

func tlsServer() {
	cert, err := tls.LoadX509KeyPair("cert.pem", "key.pem")
	if err != nil {
		log.Println(err)
		return
	}
	config := &tls.Config{Certificates: []tls.Certificate{cert}}
	ln, err := tls.Listen("tcp", ":4433", config)
	if err != nil {
		log.Println(err)
		return
	}

	defer ln.Close()
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()

	for {

		dataBuff := make([]byte, 65535)

		n, err := conn.Read(dataBuff)
		if err != nil {
			fmt.Println(err)
			time.Sleep(time.Second * 10)
			continue
		}
		fmt.Println(dataBuff[:n])
		fmt.Println(dataBuff[0])
	}
}
