package main

import (
	"fmt"
	"os"
	"runtime"
	"time"

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

	message := request.GetMessage()
	connect := request.GetConnect()
	n, err := connect.Send(message.ID(), message.Bytes())
	fmt.Println("conn.send.n", n, "send err", err, "recv len()", message.Len())

	// 以下方式都可以获取到所有连接
	// 1、request.GetConnects()
	// 2、connect.GetConnectMgr().GetConnects()

	for _, client := range request.GetConnects() {

		// 排除自己
		if client.GetID() == connect.GetID() {
			continue
		}

		// 给其它连接推送消息
		fmt.Println(client.Send(uint32(1), []byte("hello world!")))
	}

	// 主动关闭连接
	// connect.Close()
}

type UserInfoRoute struct {
}

func (u *UserInfoRoute) Do(request iface.IRequest) {
	fmt.Println("Through middleware")
	fmt.Println(request.GetMessage().Bytes())
}

//global 全局中间件
func global() iface.MiddlewareFunc {
	return func(ctx iface.IContext, next iface.Next) interface{} {

		fmt.Println("Front middleware")
		fmt.Println("ctx data", ctx.GetConnect(), ctx.GetRequest(), ctx.GetMessage())

		ctx.Set("key", "value")
		ctx.Set("now", time.Now().UnixNano())

		// 继续往下执行
		resp := next(ctx)

		fmt.Println("Rear middleware")
		return resp
	}
}

func demo() iface.MiddlewareFunc {
	return func(ctx iface.IContext, next iface.Next) interface{} {
		fmt.Println("demo middleware start")
		fmt.Printf("key=%v now=%v\n", ctx.Get("key"), ctx.Get("now"))
		resp := next(ctx)
		fmt.Println("demo middleware end")
		return resp
	}
}

var loginStore map[int]time.Time

//authentication 分组中间件
func authentication() iface.MiddlewareFunc {

	return func(ctx iface.IContext, next iface.Next) interface{} {

		conn := ctx.GetConnect()
		// 是否登录过
		if _, ok := loginStore[conn.GetID()]; !ok {
			_, _ = conn.Send(1, []byte("Authentication failed"))
			_ = conn.Close()
			return nil
		}

		return next(ctx)

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
		server.WithMaxBodyLength(0),             // 配置包体最大长度，默认为0（不限制大小）
		server.WithTCPKeepAlive(time.Second*30), // 设置TCPKeepAlive
		server.WithLogOutput(os.Stdout),         // 框架运行日志保存的地方
		//server.WithPacker() // 可自行实现数据封包解包

		// 心跳检测机制，二者需要同时配置才会生效
		server.WithHeartbeatCheckInterval(time.Second*60), // 表示60秒检测一次
		server.WithHeartbeatIdleTime(time.Second*180),     // 表示一个连接如果180秒内未向服务器发送任何数据，此连接将被强制关闭
	)

	// 全局中间件，每个路由都会执行
	s.Use(global())
	s.Use(demo())

	// 根据业务需求，添加路由
	s.AddRouter(0, new(HelloRouter))
	//s.AddRouter(1, new(XXRouter))
	// ...

	// 分组中间件
	g := s.Group(authentication())
	{
		g.AddRouter(1, new(UserInfoRoute))
		//g.AddRouter(2,new(xxx))
		//g.AddRouter(3,new(xxx))
		//g.AddRouter(4,new(xxx))
	}

	// 启动
	s.Start()
}
