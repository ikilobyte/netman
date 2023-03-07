package main

import (
	"fmt"
	"github.com/ikilobyte/netman/iface"
	"github.com/ikilobyte/netman/server"
	"os"
)

type Hooks struct {
}

func (h *Hooks) OnOpen(connect iface.IConnect) {
	fmt.Println("onOpen.udp", connect)
	fmt.Println(connect.GetAddress().String())
	fmt.Println(connect.GetAddress().Network())
	fmt.Println(connect.IsUDP())
}

func (h *Hooks) OnClose(connect iface.IConnect) {
	fmt.Println("onClose.udp", connect.GetID())
}

type Hello struct {
}

func (h *Hello) Do(request iface.IRequest) {
	fmt.Println(request.GetMessage())
}

func main() {

	fmt.Println(os.Getpid())
	s := server.UDP(
		"127.0.0.1",
		6565,
		server.WithHooks(new(Hooks)),
	)
	s.AddRouter(0, new(Hello))
	defer s.Stop()
	s.Start()
}
