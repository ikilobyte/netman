- [是什么](#是什么)
- [有什么优势](#有什么优势)
- [架构图](#架构图)
- [安装](#开始)

### 是什么
- 轻量级的高性能TCP网络框架，基于epoll/kqueue，reactor模型实现
- 支持路由配置，更专注业务需求的处理，无需关心封包解包
- 支持自定义封包格式，更灵活
- 支持linux/macos，暂不支持windows，windows请在docker中运行
- 支持TLS
- 支持websocket协议
- 支持消息中间件

### 有什么优势
- Go的net包底层也是基于epoll，但未对外开放相关epoll的处理，目的是简化开发流程
- 在业务中通常是一个连接过来后，业务层开启一个`goroutine`来处理这个连接的请求，处理大量连接的同时也会开启大量的`goroutine`
- GMP模型中如果存在大量的`goroutine`其实会影响到调度，本地P存储的G数量是有限的
- 在大多数场景下，连接建立后并不是一直都在发送消息，所以通过epoll处理后，只处理有事件的连接
- 经过测试在阿里云服务器(单机)上建立100万个连接的内存消耗在3.8G左右，连接建立后每3分钟发送一次消息到服务器并响应

### 架构图
![on](./examples/processon.png)

> windows用户查看代码可以在goland中设置一下Build Tags，这样就有 unix.* 相关的代码提示
> 
![tabs](./examples/build-tag.png)


### 开始
```bash
go get -u github.com/ikilobyte/netman
```


### tcp examples

- **server**

```go
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

```

-  **client**

```go
package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/ikilobyte/netman/util"
)

func main() {

	conn, err := net.Dial("tcp", "127.0.0.1:6565")
	if err != nil {
		panic(err)
	}
	packer := util.NewDataPacker()

	go func() {
		for {

			// 默认的封包解包规则
			header := make([]byte, 8)
			_, err := io.ReadFull(conn, header)
			if err != nil {
				fmt.Println("read head bytes err", err)
				os.Exit(1)
			}

			// 解包头部
			message, err := packer.UnPack(header)
			if err != nil {
				fmt.Println("unpack err", err)
				os.Exit(1)
			}

			// 创建一个和数据大小一样的bytes并读取
			dataBuff := make([]byte, message.Len())
			n, err := io.ReadFull(conn, dataBuff)
			if err != nil {
				fmt.Println("read dataBuff err", err, len(dataBuff[:n]))
				os.Exit(1)
			}
			message.SetData(dataBuff)

			fmt.Printf(
				"recv msgID[%d] len[%d] %s \n",
				message.ID(),
				message.Len(),
				time.Now().Format("2006-01-02 15:04:05.0000"),
			)
		}
	}()

	// 100MB
	c := strings.Repeat("a", 1024*1024*100)
    bs, err := packer.Pack(0, []byte(c))
    if err != nil {
        panic(err)
    }
    
	for {
		fmt.Println(conn.Write(bs))
		time.Sleep(time.Second * 2)
	}
}
```


### websocket

- **server**
```go
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
	fmt.Println("onopen", connect.GetID())
}

func (h *Handler) Message(request iface.IRequest) {

	// 消息
	message := request.GetMessage()

	// 来自那个连接的
	connect := request.GetConnect()

	fmt.Printf("recv %s\n", message.String())

	// 普通文本格式
	fmt.Println(connect.Text([]byte(fmt.Sprintf("hi %s", message.Bytes()))))

	// 二进制格式
	//fmt.Println(connect.Binary([]byte("hi")))
}

func (h *Handler) Close(connect iface.IConnect) {
	fmt.Println("onclose", connect.GetID())
}

func main() {

	fmt.Println(os.Getpid())

	// 构造
	s := server.Websocket("0.0.0.0", 6565,
		new(Handler), // websocket事件回调处理
		server.WithNumEventLoop(runtime.NumCPU()*3), // 配置reactor线程的数量
		server.WithTCPKeepAlive(time.Second*30),     // 设置TCPKeepAlive
		server.WithLogOutput(os.Stdout),             // 框架运行日志保存的地方

		// 心跳检测机制，二者需要同时配置才会生效
		server.WithHeartbeatCheckInterval(time.Second*60), // 表示60秒检测一次
		server.WithHeartbeatIdleTime(time.Second*180),     // 表示一个连接如果180秒内未向服务器发送任何数据，此连接将被强制关闭
	)

	// 启动
	s.Start()
}

```

- **client**
```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>websocket client</title>

    <style>
        .container {
            display: flex;
            flex-direction: row;
        }

        .container > div {
            margin-right: 2rem;
        }
    </style>
</head>
<body>

<div class="container">
    <div><button id="connect">连接</button></div>
    <div><button id="send">发送消息</button></div>
    <div><button id="close">关闭连接</button></div>
</div>
<div id="append"></div>

<script>

    var ws;
    document.getElementById('connect').addEventListener('click',() => {
        ws = new WebSocket('ws://127.0.0.1:6565')
        ws.onopen = function(e){
            console.log('onopen',e)
        }

        ws.onmessage = function(e){
            let p = document.createElement('p')
            p.innerHTML = e.data;
            document.getElementById('append').appendChild(p)
        }

        ws.onclose = function(e){
            console.log('onclose',e)
        }

        ws.onerror = function(e){
            console.log('onerror',e)
        }
    })

    document.getElementById('send').addEventListener('click',() => {
        if(!ws){
            console.error('请先连接');
            return ;
        }
        let data = {
            key: 'value',
            now: Date.now()
        }
        ws.send(JSON.stringify(data))
    })


    document.getElementById('close').addEventListener('click',() => {
        if(!ws){
            console.error('请先连接');
            return ;
        }
        ws.close()
    })

</script>
</body>
</html>
```

> 以下示例为TLS示例，如无TLS需求，可忽略

### TLS examples

- **server**

```go
package main

import (
	"crypto/tls"
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
	msg := request.GetMessage()
	connect := request.GetConnect()
	n, err := connect.Send(1, msg.Bytes())
	fmt.Println("conn.Send.n", n, "Send.error", err)

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

	// 关闭连接
	// connect.Close()
}

func main() {

	fmt.Println(os.Getpid())

	// 配置tls
	pair, err := tls.LoadX509KeyPair("./server.pem", "./server.key")
	if err != nil {
		panic(err)
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{pair},
	}

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

		// 两个同时存在时，使用WithTLSConfig的配置
		// 开启TLS（后续版本将删除）
		server.WithTls("./server.pem", "./server.key"),

		// 开启TLS（推荐使用）
		server.WithTLSConfig(tlsConfig),
	)

	// 根据业务需求，添加路由
	s.AddRouter(0, new(HelloRouter))
	//s.AddRouter(1, new(XXRouter))
	// ...

	// 启动
	s.Start()
}

```


- **client**

```go

package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/ikilobyte/netman/util"
)

func main() {

	conn, err := tls.Dial("tcp", "127.0.0.1:6565", &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		panic(err)
	}
	packer := util.NewDataPacker()

	go func() {
		for {

			// 默认的封包解包规则
			header := make([]byte, 8)
			_, err := io.ReadFull(conn, header)
			if err != nil {
				fmt.Println("read head bytes err", err)
				os.Exit(1)
			}

			// 解包头部
			message, err := packer.UnPack(header)
			if err != nil {
				fmt.Println("unpack err", err)
				os.Exit(1)
			}

			// 创建一个和数据大小一样的bytes并读取
			dataBuff := make([]byte, message.Len())
			n, err := io.ReadFull(conn, dataBuff)
			if err != nil {
				fmt.Println("read dataBuff err", err, len(dataBuff[:n]))
				os.Exit(1)
			}
			message.SetData(dataBuff)

			fmt.Printf(
				"recv msgID[%d] len[%d] %s \n",
				message.ID(),
				message.Len(),
				time.Now().Format("2006-01-02 15:04:05.0000"),
			)
		}
	}()

	// 2MB 原始数据
	c := strings.Repeat("a", 1024*1024*2)
	for {
		bs, err := packer.Pack(0, []byte(c))
		if err != nil {
			panic(err)
		}
		fmt.Println(conn.Write(bs))
		time.Sleep(time.Second * 1)
	}
}
```


### 百万连接测试结果
* 如看不到图片可以在`examples`目录中查看`c1000k.png`这张图片
![c1000k](./examples/c1000k.png)