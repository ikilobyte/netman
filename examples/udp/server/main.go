package main

import (
	"fmt"
	"github.com/ikilobyte/netman/iface"
	"github.com/ikilobyte/netman/server"
	"os"
	"time"
)

type Hooks struct {
}

func (h *Hooks) OnOpen(connect iface.IConnect) {
	fmt.Printf("udp onopen id@%d fd@%d\n", connect.GetID(), connect.GetFd())
}

//OnClose 是的，UDP也抽象出了 onClose
func (h *Hooks) OnClose(connect iface.IConnect) {
	fmt.Printf("udp onclose id@%d fd@%d\n", connect.GetID(), connect.GetFd())
}

type Hello struct {
}

func (h *Hello) Do(request iface.IRequest) {
	conn := request.GetConnect()
	fmt.Println("onPacket", request.GetMessage().Bytes())
	fmt.Println(conn.Send(0, []byte("hello reply")))
}

func main() {

	fmt.Println(os.Getpid())
	s := server.UDP(
		"127.0.0.1",
		6565,
		server.WithHooks(new(Hooks)),
		server.WithUDPPacketBufferLength(32767), // UDP一次收包最大长度，请合理配置

		// 推荐
		server.WithHeartbeatIdleTime(time.Second*5),
		server.WithHeartbeatCheckInterval(time.Second*10),
	)

	// 中间使用参考tcp server示例，支持全局中间件、路由中间件 是通用的
	s.AddRouter(0, new(Hello))
	defer s.Stop()
	s.Start()
}
