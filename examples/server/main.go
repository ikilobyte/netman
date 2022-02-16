package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/ikilobyte/netman/iface"

	"github.com/ikilobyte/netman/server"
)

type Hooks struct{}

func (h *Hooks) OnOpen(connect iface.IConnect) {
	fmt.Printf("connId[%d] onOpen\n", connect.GetID())
	//fmt.Println(unix.GetsockoptInt(connect.GetFd(), unix.SOL_SOCKET, unix.SO_SNDBUF))
}

func (h *Hooks) OnClose(connect iface.IConnect) {
	fmt.Printf("connId[%d] onClose\n", connect.GetID())
}

type HelloRouter struct{}

func (h *HelloRouter) Do(request iface.IRequest) {
	conn := request.GetConnect()
	msg := request.GetMessage()
	conn.Write(msg.ID(), msg.Bytes())
	conn.Write(101, []byte("hello world"))
	conn.Write(102, []byte("你好"))
	conn.Write(103, []byte("你好"))
	conn.Write(104, []byte("你好"))
	conn.Write(105, []byte("你好"))
	conn.Write(106, []byte("你好"))
	conn.Write(107, []byte("你好"))
}

func main() {

	fmt.Println(os.Getpid())

	// 构造
	s := server.New(
		"0.0.0.0",
		6565,
		server.WithNumEventLoop(runtime.NumCPU()*3),
		server.WithHooks(new(Hooks)), // hook
		//server.WithPacker() // 可自行实现数据封包解包
	)

	// 根据业务需求，添加路由
	s.AddRouter(0, new(HelloRouter))
	//s.AddRouter(1, new(XXRouter))
	// ...

	// 启动
	s.Start()
}
