package server

import (
	"net"

	"github.com/ikilobyte/netman/iface"
	"golang.org/x/sys/unix"
)

//Connect TCP连接成功建立后，会抽象一个Connect
type Connect struct {
	id      int           // 自定义生成的ID
	fd      int           // 系统分配的fd
	epfd    int           // 管理这个连接的epoll
	packer  iface.IPacker // 封包解包实现，可以自行实现
	Address net.Addr
	hooks   iface.IHooks
}

//NewConnect 构造一个连接
func newConnect(id int, fd int, address net.Addr, packer iface.IPacker, hooks iface.IHooks) *Connect {
	connect := &Connect{
		id:      id,
		fd:      fd,
		packer:  packer,
		Address: address,
		hooks:   hooks,
	}

	// 执行回调
	if hooks != nil {
		go hooks.OnOpen(connect)
	}

	return connect
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
	err := unix.Close(c.fd)
	// 关闭成功才执行
	if c.hooks != nil && err == nil {
		c.hooks.OnClose(c)
	}
	return err
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

func (c *Connect) GetAddress() net.Addr {
	return c.Address
}

//Write 写数据
func (c *Connect) Write(msgID uint32, bytes []byte) (int, error) {

	// 1、封包
	dataPack, err := c.packer.Pack(msgID, bytes)
	if err != nil {
		return 0, err
	}

	// 2、发送
	return unix.Write(c.fd, dataPack)
}

//SetEpFd 设置这个连接属于哪个epoll
func (c *Connect) SetEpFd(epfd int) {
	c.epfd = epfd
}

//GetEpFd 获取这个连接的epoll fd
func (c *Connect) GetEpFd() int {
	return c.epfd
}
