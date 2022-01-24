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
