# 目录
* [目录](#目录)
    * [介绍](#介绍)
    * [优势](#优势)
    * [安装](#安装)
    * [开始](#开始)
    * [Websocket](#Websocket)
    * [中间件](#中间件)
    * [配置](#配置)
        * [Hooks](#Hooks)
        * [心跳](#心跳检测)
        * [包体最大长度](#包体最大长度)
        * [TCP Keepalive](#tcp-keepalive)
        * [TLS](#TLS)
        * [自定义封包解包](#自定义封包解包)
        * [组合使用](#组合使用)
    * [架构](#架构)
    * [百万连接](#百万连接)

## 介绍
- 轻量的高性能TCP网络框架，基于epoll/kqueue，reactor模型实现
- 简单的API，细节在框架内部实现，几行代码即可构建高性能的Server
- 支持路由配置，更专注业务需求的处理，无需关心封包解包
- 支持自定义封包格式，更灵活
- 支持linux/macos，windows请在docker中运行
- 支持TLS
- 支持websocket
- 中间件

## 优势
- 非阻塞IO
- 底层基于事件循环，在`net`包中，一个连接需要一个`goroutine`去维持，但`netman`基于事件循环则不需要，大大减少了内存的占用，在大量连接的场景下更为明显
- 基于路由配置，业务层不关心封包解包的实现
- 全局中间件、分组中间件
- 经过测试在阿里云服务器(单机)上建立100万个连接（C1000K）的内存消耗在`3.8GB`左右

## 安装
* 下载

    ```bash
    go get -u github.com/ikilobyte/netman
    ```
* 导入
    ```go
    import "github.com/ikilobyte/netman/server"
    ```

## 开始
### server

* 基本使用
    ```go
    package main

    import "github.com/ikilobyte/netman/server"
    
    type Hello struct{}

    func (h *Hello) Do(request iface.IRequest) {
        message := request.GetMessage()
        connect := request.GetConnect()
        n, err := connect.Send(message.ID(), message.Bytes())
        fmt.Println("conn.send.n", n, "send err", err, "recv len()", message.Len())
        
        // 以下方式都可以获取到所有连接
        // 1、request.GetConnects()
        // 2、connect.GetConnectMgr().GetConnects()
        
        // 主动关闭连接
        // connect.Close()
    }
    
    func main() {
	    s := server.New(
	        "0.0.0.0",
	        6565,
            
	        // options 更多配置请看 #配置 文档
	        server.WithMaxBodyLength(1024*1024*100), // 包体最大长度限制，0表示不限制
	    )
	    
	    // add router 
		s.AddRouter(0, new(Hello))  // 设置消息ID为0的处理方法
        //s.AddRouter(1, new(xxx))  // ...
        
	    s.Start()
    }
    ```

### client
* 示例
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
        
        // 用于消息的封包和解包，也可以自行实现封包解包规则
        packer := util.NewDataPacker()
        
        // 100MB
        c := strings.Repeat("a", 1024*1024*100)
        bs, err := packer.Pack(0, []byte(c))
        
        if err != nil {
            panic(err)
        }
        
        // 发送消息
        for {
            fmt.Println(conn.Write(bs))
            time.Sleep(time.Second * 1)
        }
    }
    
* 接收消息
    ```go
    // 备注：以下规则是框架默认实现的规则，你也可以自行实现，使用自己的 Packer 即可
    for {
        header := make([]byte, 8)
        n, err := io.ReadFull(conn, header)
        if n == 0 && err == io.EOF {
        	fmt.Println("连接已断开")
        	os.Exit(0)
        }
        
        if err != nil {
        	fmt.Println("read head bytes err", err)
        	os.Exit(1)
        }
        
        // 解包头部，会返回一个IMessage
        message, err := packer.UnPack(header)
        if err != nil {
        	fmt.Println("unpack err", err)
        	os.Exit(1)
        }
        
        // 创建一个和数据大小一样的bytes并读取
        dataBuff := make([]byte, message.Len())
        n, err = io.ReadFull(conn, dataBuff)
        
        if n == 0 && err == io.EOF {
        	fmt.Println("连接已断开")
        	os.Exit(0)
        }
        
        if err != nil {
        	fmt.Println("read dataBuff err", err, len(dataBuff[:n]))
        	os.Exit(1)
        }
        
        message.SetData(dataBuff)
        fmt.Printf(
            "recv msgID[%d] len[%d] %s\n",
            message.ID(),
            message.Len(),
            time.Now().Format("2006-01-02 15:04:05.000"),
        )
    }
    
## Websocket
* server
```go
type Handler struct{}

// 连接建立
func (h *Handler) Open(connect iface.IConnect) {
	fmt.Println("onopen", connect.GetID())
}

// 消息到来时
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

// 连接关闭
func (h *Handler) Close(connect iface.IConnect) {
	fmt.Println("onclose", connect.GetID())
}

s := server.Websocket(
    "0.0.0.0",
    6565,
    new(Handler),   // websocket事件回调处理
)
```

* client
* 各语言的Websocket Client库即可，如Javascript的 `new Websocket`
* [`client.html`](./examples/websocket/client.html)

## 中间件
* 可被定义为`全局中间件`，和`分组中间件`，目前websocket只支持`全局中间件`
* 配置中间件后，接收到的每条消息都会先经过中间件，再到达对应的消息回调函数
* 中间件可提前终止执行
* 定义中间件
    ```go
    func demo1() iface.MiddlewareFunc {
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
    
    func demo2() iface.MiddlewareFunc {
        return func(ctx iface.IContext, next iface.Next) interface{} {
            fmt.Println(ctx.Get("key"),ctx.Get("now"))
            return next(ctx)
        }
    }   
    
    //authentication 这个用来做分组中间件
    var loginStore map[int]time.Time
    func authentication() iface.MiddlewareFunc {
        return func(ctx iface.IContext, next iface.Next) interface{} {
            conn := ctx.GetConnect()
            // 判断是否登录过，
            if _, ok := loginStore[conn.GetID()]; !ok {
                // 提前结束执行，不会到对应的Router
                _, _ = conn.Send(1, []byte("Authentication failed"))
                _ = conn.Close()
                return nil
            }
            // 继续执行
            return next(ctx)
        }
    }
    
    ``` 
* 使用
    ```go
    // 全局中间件
    s.Use(demo1())
    s.Use(demo2())
    
    // 分组，只有对应的路由才会执行
    g := s.Group(authentication())
	{
        g.AddRouter(1, new(xxx))
        //g.AddRouter(2,new(xxx))
        //g.AddRouter(3,new(xxx))
        //g.AddRouter(4,new(xxx))
	}
    ```

## 配置
* 所有配置对 `TcpServer（TLS）`、`Websocket Server` 都是生效的
* 更多配置请查看 [`options.go`](./server/options.go)

### Hooks
* Websocket也可以生效
```go
type Hooks struct{}

// OnOpen 连接建立后回调
func (h *Hooks) OnOpen(connect iface.IConnect) {
    fmt.Printf("connId[%d] onOpen\n", connect.GetID())
}

// OnClose 连接成功关闭时回调
func (h *Hooks) OnClose(connect iface.IConnect) {
    fmt.Printf("connId[%d] onClose\n", connect.GetID())
}

s := server.New(
    "0.0.0.0",
    6565,
    
    // 配置 Hooks
    server.WithHooks(new(Hooks)),
)
```

### 心跳检测
* 二者需要同时配置才会生效
```go
s := server.New(
    "0.0.0.0",
    6565,
    
    // 表示60秒检测一次
    server.WithHeartbeatCheckInterval(time.Second*60), 
    
    // 表示一个连接如果180秒内未向服务器发送任何数据，此连接将被强制关闭
    server.WithHeartbeatIdleTime(time.Second*180),     
)
```

### 包体最大长度
```go
s := server.New(
    "0.0.0.0",
    6565,
    
    // 0表示不限制长度
    // 这里配置的是100MB，当某条消息超过100MB时，会被拒绝处理
    server.WithMaxBodyLength(1024*1024*100),
)
```

### TCP Keepalive
* 参考：https://zh.wikipedia.org/wiki/Keepalive
```go
s := server.New(
    "0.0.0.0",
    6565,
    
    server.WithTCPKeepAlive(time.Second*30),
)
```

### TLS
```go
tlsConfig := &tls.Config{
    Certificates: []tls.Certificate{...},
}

s := server.New(
    "0.0.0.0",
    6565,
    
    // 传入相关配置后，即可开启TLS
    server.WithTLSConfig(tlsConfig),
)
```


### 自定义封包解包
* 为了更灵活的需求，可自定义封包解包规则，只需要使用`IPacker`接口即可
* 配置
```go
// IPacker 定义
type IPacker interface {
    Pack(msgID uint32, data []byte) ([]byte, error) // 封包
    UnPack([]byte) (IMessage, error)                // 解包
    SetMaxBodyLength(uint32)                        // 设置包体最大长度限制
    GetHeaderLength() uint32                        // 获取头部长度
}

type YouPacker struct {
    // implements IPacker
    // ... 
}
    
s := server.New(
    "0.0.0.0",
    6565,
    
    // 自定义Packer
    server.server.WithPacker(new(YouPacker)),
)
```
### 组合使用
```go
s := server.New(
    "0.0.0.0",
    6565,
    server.WithNumEventLoop(runtime.NumCPU()*3),
    server.WithHooks(new(Hooks)),            // hook
    server.WithMaxBodyLength(0),             // 配置包体最大长度，默认为0（不限制大小）
    server.WithTCPKeepAlive(time.Second*30), // 设置TCPKeepAlive
    server.WithLogOutput(os.Stdout),         // 框架运行日志保存的地方
    server.WithPacker(new(YouPacker)),       // 可自行实现数据封包解包
    
    // 心跳检测机制，二者需要同时配置才会生效
    server.WithHeartbeatCheckInterval(time.Second*60), // 表示60秒检测一次
    server.WithHeartbeatIdleTime(time.Second*180),     // 表示一个连接如果180秒内未向服务器发送任何数据，此连接将被强制关闭
    
    // 开启TLS
    server.WithTLSConfig(tlsConfig),
)

s.Start()
```

## 架构
![on](./examples/processon.png)

## 百万连接
* 如看不到图片可以在`examples`目录中查看`c1000k.png`这张图片
  ![c1000k](./examples/c1000k.png)

## 鸣谢
感谢 JetBrains 为此开源项目提供 GoLand 开发工具支持：

<img src="https://resources.jetbrains.com/storage/products/company/brand/logos/GoLand.svg" width="300">