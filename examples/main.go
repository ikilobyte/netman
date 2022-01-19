package main

import (
	"github.com/ikilobyte/netman/server"
)

func main() {
	ser := server.New("0.0.0.0", 6006)
	ser.Start()
}
