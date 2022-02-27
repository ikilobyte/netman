package main

import (
	"fmt"
	"os"
	"runtime"
	"time"

	stdtls "github.com/ikilobyte/netman/std/tls"

	"github.com/ikilobyte/netman/server"

	"github.com/ikilobyte/netman/iface"
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
	msg := request.GetMessage()
	fmt.Println(msg.String())

	//conn.Send(msg.ID(), msg.Bytes())
	//for i := 0; i < 50; i++ {
	//	conn.Send(uint32(i), []byte("hello world"))
	//}
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

		// 二者需要同时配置才会生效
		server.WithHeartbeatCheckInterval(time.Second*60), // 表示60秒检测一次
		server.WithHeartbeatIdleTime(time.Second*180),     // 表示一个连接如果180秒内未向服务器发送任何数据，此连接将被强制关闭
		server.WithTls("server.pem", "server.key"),
		//server.WithPacker() // 可自行实现数据封包解包
	)

	// 根据业务需求，添加路由
	s.AddRouter(0, new(HelloRouter))
	//s.AddRouter(1, new(XXRouter))
	// ...

	// 启动
	s.Start()
}

func stdTlsServer() {
	pair, err := stdtls.LoadX509KeyPair("server.pem", "server.key")
	if err != nil {
		panic(err)
	}
	conf := &stdtls.Config{Certificates: []stdtls.Certificate{pair}}
	server.WithTls("server.pem", "server.key")
	listener, err := stdtls.Listen("tcp", ":6565", conf)
	if err != nil {
		panic(err)
	}
	for {
		conn, err := listener.Accept()

		if err != nil {
			continue
		}

		go func() {
			total := 0
			for {
				dataBuff := make([]byte, 128)
				n, err := conn.Read(dataBuff)
				if n == 0 {
					conn.Close()
					fmt.Println("已断开连接")
					return
				}
				if err != nil {
					fmt.Println(err)
				}
				total += 1
				fmt.Println("读取到数据", n, string(dataBuff[:n]), total)

				time.Sleep(time.Hour)
			}
		}()
	}
}
