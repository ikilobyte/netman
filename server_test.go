package netman

import (
	"fmt"
	"os"
	"testing"
)

func TestServer(t *testing.T) {
	fmt.Println("os.Getpid()", os.Getpid())
	server := NewServer(
		"127.0.0.1",
		6100,
		WithNumEventLoop(5),
		WithNumWorker(10), // 设置多少个goroutine处理业务逻辑
	)

	server.Start()
}
