package main

import (
	"fmt"

	"github.com/ikilobyte/netman/server"
)

func main() {

	ser := server.New(
		"0.0.0.0",
		6003,
		server.WithNumEventLoop(10),
		server.WithNumWorker(30),
	)
	ser.Start()
	fmt.Println("wqrqwr", ser)
}
