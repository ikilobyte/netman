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
