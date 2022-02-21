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

	fmt.Println("os.Getpid()", os.Getpid())
	packer := util.NewDataPacker()

	go func() {
		for {

			// 默认的封包解包规则
			header := make([]byte, 8)
			_, err := io.ReadFull(conn, header)
			if err != nil {
				fmt.Println("read head err", err)
				os.Exit(1)
				time.Sleep(time.Hour)
				continue
			}

			// 解包头部
			message, err := packer.UnPack(header)
			if err != nil {
				fmt.Println("unpack err", err)
				continue
			}

			// 创建一个和数据大小一样的bytes并读取
			//fmt.Println("message.len", message.Len(), header, "正在读取这么多")
			dataBuff := make([]byte, message.Len())
			n, err := io.ReadFull(conn, dataBuff)
			if err != nil {
				fmt.Println("read dataBuff err", err, len(dataBuff[:n]))
				os.Exit(1)
				continue
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
	id := uint32(0)
	for {
		fmt.Printf("第%d轮正在发送数据 %q\n", id, time.Now().String())
		s := time.Now()
		bs, err := packer.Pack(0, []byte(c))
		if err != nil {
			panic(err)
		}
		n, err := conn.Write(bs)

		fmt.Printf("已发送：%d 耗时：%q total：%d\n", n, time.Since(s), id)
		time.Sleep(time.Second * 2)
		id += 1
	}

}
