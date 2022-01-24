### 是什么
- 轻量级的高性能TCP网络框架，基于epoll/kqueue，reactor模型实现
- 支持路由配置，更专注业务需求的处理，无需关心封包解包
- 支持自定义封包格式，更灵活

### 开始
```bash
go get github.com/ikilobyte/netman
```


### server端
```go
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
  conn := request.GetConnect()
  msg := request.GetMessage()
  fmt.Println("recv", msg.String())
  conn.Write(msg.ID(), []byte(fmt.Sprintf("server resp %s", msg.String())))
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

  // 根据业务需求，添加路由
  s.AddRouter(0, new(HelloRouter))
  //s.AddRouter(1, new(XXRouter))
  // ...

  // 启动
  s.Start()
}
```


### client端
```go
package main

import (
	"fmt"
	"io"
	"net"
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
				fmt.Println("read head err", err)
				continue
			}

			// 解包头部
			message, err := packer.UnPack(header)
			if err != nil {
				fmt.Println("unpack err", err)
				continue
			}

			// 创建一个和数据大小一样的bytes并读取
			dataBuff := make([]byte, message.Len())
			_, err = io.ReadFull(conn, dataBuff)
			if err != nil {
				fmt.Println("read dataBuff err", err)
				continue
			}
			message.SetData(dataBuff)

			fmt.Printf("recv msgID[%d] len[%d] %q \n", message.ID(), message.Len(), message.String())
		}
	}()

	for {

		bs, err := packer.Pack(0, []byte(fmt.Sprintf("hello netMan %s", time.Now().Format("2006-01-02 15:04:05.0000"))))
		if err != nil {
			panic(err)
		}
		conn.Write(bs)
		time.Sleep(time.Second * 1)
	}

}
```