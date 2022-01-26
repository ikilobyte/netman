package main

import (
	"fmt"
	"io"
	"net"
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
				fmt.Println("read head err", err)
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
			dataBuff := make([]byte, message.Len())
			n, err := io.ReadFull(conn, dataBuff)
			if err != nil {
				fmt.Println("read dataBuff err", err, len(dataBuff[:n]))
				time.Sleep(time.Hour * 30)
				continue
			}
			message.SetData(dataBuff)

			fmt.Printf("recv msgID[%d] len[%d]  \n", message.ID(), message.Len())
		}
	}()

	// 10MB
	c := strings.Repeat("a", 1024*1024*10)
	//c := strings.Repeat("a", 1024)

	for {
		fmt.Println("start send")
		bs, err := packer.Pack(0, []byte(fmt.Sprintf("%s", c)))
		if err != nil {
			panic(err)
		}

		// 写入大量数据
		n, err := conn.Write(bs)
		fmt.Println("end send", n, err, time.Now().Format("2006-01-02 15:04:05.0000"))
		time.Sleep(time.Second * 1000)
	}

}
