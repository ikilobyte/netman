package main

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/ikilobyte/netman/server"

	"github.com/ikilobyte/netman/iface"
)

type Handler struct{}

func (h *Handler) Open(connect iface.IConnect) {

	// 获取query参数
	//query := connect.GetQueryStringParam()

	//fmt.Println(query)
	//if query.Get("token") != "xxx" {
	//	// 关闭连接
	//	connect.Close()
	//	return
	//}
	//fmt.Println("onopen", connect.GetID())
}

func (h *Handler) Message(request iface.IRequest) {

	// 消息
	message := request.GetMessage()

	// 来自那个连接的
	connect := request.GetConnect()

	// 判断是什么消息类型
	if message.IsText() {
		fmt.Println(connect.Text(message.Bytes()))
	} else {
		fmt.Println(connect.Binary(message.Bytes()))
	}
}

func (h *Handler) Close(connect iface.IConnect) {
	fmt.Println("onclose", connect.GetID())
}

//log 定义中间件
func log() iface.MiddlewareFunc {
	return func(ctx iface.IContext, next iface.Next) interface{} {
		fmt.Printf(
			"log middleware connID %v message[%v] now %s\n",
			ctx.GetConnect().GetID(),
			ctx.GetMessage().String(),
			time.Now().Format("2006-01-02 15:04:05.000"),
		)
		return next(ctx)
	}
}

func main() {

	fmt.Println(os.Getpid())

	// 构造
	s := server.Websocket(
		"0.0.0.0",
		6565,
		new(Handler), // websocket事件回调处理
		server.WithNumEventLoop(runtime.NumCPU()), // 配置reactor线程的数量
		server.WithTCPKeepAlive(time.Second*30),   // 设置TCPKeepAlive
		server.WithLogOutput(os.Stdout),           // 框架运行日志保存的地方

		// 心跳检测机制，二者需要同时配置才会生效
		server.WithHeartbeatCheckInterval(time.Second*60), // 表示60秒检测一次
		server.WithHeartbeatIdleTime(time.Second*180),     // 表示一个连接如果180秒内未向服务器发送任何数据，此连接将被强制关闭
	)

	// 全局中间件
	//s.Use(log())
	//s.Use(xxx)

	// 启动
	s.Start()
}
