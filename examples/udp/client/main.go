package main

import (
	"fmt"
	"github.com/ikilobyte/netman/util"
	"net"
	"os"
	"sync"
	"time"
)

func main() {
	fmt.Println(os.Getpid())
	wg := &sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go connect(i, wg)
	}
	wg.Wait()
}

func connect(id int, wg *sync.WaitGroup) {
	defer wg.Done()
	conn, err := net.Dial("udp", "127.0.0.1:6565")
	if err != nil {
		fmt.Printf("id dail udp err %v\n", err)
		return
	}

	packer := util.NewDataPacker()
	packet, _ := packer.Pack(0, []byte(fmt.Sprintf("from %d hello udp server", id)))
	headLen := int(packer.GetHeaderLength())
	for {

		// 发送数据
		_, err := conn.Write(packet)
		if err != nil {
			fmt.Printf("id@%d write err %v\n", id, err)
			return
		}

		// 读取数据
		buffer := make([]byte, 1024)
		_, err = conn.Read(buffer)
		if err != nil {
			fmt.Printf("id@%d read err %v\n", id, err)
			return
		}

		message, err := packer.UnPack(buffer)
		if err != nil {
			fmt.Printf("id@%d unpack err %v\n", id, err)
		}

		message.SetData(buffer[headLen : headLen+message.Len()])
		fmt.Printf("id@%d recv from server %s\n", id, message.String())
		time.Sleep(time.Second * 2)
	}
}
