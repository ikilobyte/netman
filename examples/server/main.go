package main

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/ikilobyte/netman/iface"

	"github.com/ikilobyte/netman/server"
)

type Hooks struct{}

func (h *Hooks) OnOpen(connect iface.IConnect) {
	fmt.Printf("connId[%d] onOpen\n", connect.GetID())

}

func (h *Hooks) OnClose(connect iface.IConnect) {
	fmt.Printf("connId[%d] onClose\n", connect.GetID())
}

type HelloRouter struct{}

func (h *HelloRouter) Do(request iface.IRequest) {
	conn := request.GetConnect()
	msg := request.GetMessage()
	conn.Write(msg.ID(), msg.Bytes())
	for i := 0; i < 50; i++ {
		conn.Write(uint32(i), []byte("hello world"))
	}
}

func main() {

	fmt.Println(os.Getpid())

	// 构造
	s := server.New(
		"0.0.0.0",
		6565,
		server.WithNumEventLoop(runtime.NumCPU()*3),
		server.WithHooks(new(Hooks)),            // hook
		server.WithMaxBodyLength(65535),         // 配置包体最大长度，默认为0（不限制大小）
		server.WithTCPKeepAlive(time.Second*30), // 设置TCPKeepAlive
		//server.WithPacker() // 可自行实现数据封包解包
	)

	// 根据业务需求，添加路由
	s.AddRouter(0, new(HelloRouter))
	//s.AddRouter(1, new(XXRouter))
	// ...

	// 启动
	s.Start()
}
