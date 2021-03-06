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
			n, err := io.ReadFull(conn, header)
			if n == 0 && err == io.EOF {
				fmt.Println("连接已断开")
				os.Exit(0)
			}

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

			output := ""
			if message.Len() < 1024 {
				output = message.String()
			}
			fmt.Printf(
				"recv msgID[%d] len[%d] %s %s\n",
				message.ID(),
				message.Len(),
				output,
				time.Now().Format("2006-01-02 15:04:05.0000"),
			)
		}
	}()

	for i := 0; i < 10; i++ {
		go func(idx int) {

			for {
				bs, _ := packer.Pack(0, []byte(fmt.Sprintf("hello netman %d", idx)))
				b, err := conn.Write(bs)
				if err != nil {
					return
				}
				fmt.Println("idx", idx, "write.n", b)
				time.Sleep(time.Second)
			}
		}(i)
	}

	// 100MB
	c := strings.Repeat("a", 1024*1024*100)
	bs, err := packer.Pack(0, []byte(c))
	if err != nil {
		panic(err)
	}

	for {
		fmt.Println(conn.Write(bs))
		time.Sleep(time.Second * 3)
	}
}
