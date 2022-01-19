package server

import (
	"github.com/ikilobyte/netman/iface"
	"golang.org/x/sys/unix"
)

//Connect 每个连接的具体定义
type Connect struct {
	id     int // 自定义生成的ID
	fd     int // 系统分配的fd
	packer iface.IPacker
}

//NewConnect 构造一个连接
func NewConnect(id int, fd int, packer iface.IPacker) *Connect {
	return &Connect{id: id, fd: fd, packer: packer}
}

//GetID 获取连接ID
func (c *Connect) GetID() int {
	return c.id
}

//GetFd 获取系统分配的fd
func (c *Connect) GetFd() int {
	return c.fd
}

//Close 断开连接
func (c *Connect) Close() error {
	return unix.Close(c.fd)
}

// Read 读取数据
func (c *Connect) Read(bs []byte) (int, error) {
	n, err := unix.Read(c.fd, bs)
	if err != nil {
		return n, err
	}

	// 连接已断开，读取的字节是0，err==nil
	if n == 0 {
		return 0, nil
	}

	return n, nil
}

//GetPacker 获取packer
func (c *Connect) GetPacker() iface.IPacker {
	return c.packer
}
