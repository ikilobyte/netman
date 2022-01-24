package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/ikilobyte/netman/iface"

	"github.com/ikilobyte/netman/server"
)

type HelloRouter struct {
	server.BaseRouter
}

func (h *HelloRouter) Do(request iface.IRequest) {
	fmt.Println(request.GetConnect()) // 来自谁
	fmt.Println(request.GetMessage()) // 收到的消息
}

func main() {

	fmt.Println(os.Getpid())

	// 构造
	s := server.New(
		"0.0.0.0",
		6565,
		server.WithNumEventLoop(runtime.NumCPU()*3),
		//server.WithPacker() // 可自行实现数据封包解包
	)

	// 添加路由
	s.AddRouter(0, new(HelloRouter))
	//s.AddRouter(1, new(HelloRouter))
	// ...

	// 启动
	s.Start()
}
