package server

import (
	"fmt"
	"net"

	"github.com/ikilobyte/netman/util"

	"github.com/ikilobyte/netman/iface"
	"golang.org/x/sys/unix"
)

//Connect TCP连接成功建立后，会抽象一个Connect
type Connect struct {
	id        int           // 自定义生成的ID
	fd        int           // 系统分配的fd
	epfd      int           // 管理这个连接的epoll
	packer    iface.IPacker // 封包解包实现，可以自行实现
	Address   net.Addr      //
	hooks     iface.IHooks  //
	writeBuff []byte        // 待发送的数据缓冲，如果这个变为空，那就表示这一次的全部发送完毕了！
	poller    iface.IPoller //
	writeQ    *util.Queue   // TODO 写队列
}

//NewConnect 构造一个连接
func newConnect(id int, fd int, address net.Addr, packer iface.IPacker, hooks iface.IHooks) *Connect {
	connect := &Connect{
		id:      id,
		fd:      fd,
		packer:  packer,
		Address: address,
		hooks:   hooks,
		writeQ:  util.NewQueue(), // 待发送的数据队列
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
	totalBytes := len(dataPack)
	n, err := unix.Write(c.fd, dataPack)

	// err会有多种情况
	// 1、缓冲区满，无法写入，会返回	err = unix.EAGAIN
	// 2、客户端连接已断开，一般来说内核会延迟一会给出对应的err(unix.EPIPE)
	// TODO 多次写入大量数据，需要引入队列，依次取出待发送的数据
	// 这种情况一般只有发送大量(MB)数据时才会出现
	if n != totalBytes && n > 0 {
		// 同时只能存在一个状态，要么可读，要么可写，禁止并行多个状态，可以把epoll理解为状态机
		_ = c.poller.ModWrite(c.fd, c.id) // 注册可写事件，内核通知可写后，继续写入数据
		// 把剩下的保存到写入队列中
		c.writeQ.Push(dataPack[n:])
		//c.writeBuff = dataPack[n:]        // 暂存现场，下次可写时继续写这里的数据
		return totalBytes, nil
	}

	// 一个字节都未发送出去，把打包好的数据放入到写入队列中
	if n < 0 {
		c.writeQ.Push(dataPack)
	}

	fmt.Println("c.writeQ.Len()", c.writeQ.Len())
	return n, err
}

//SetEpFd 设置这个连接属于哪个epoll
func (c *Connect) SetEpFd(epfd int) {
	c.epfd = epfd
}

//GetEpFd 获取这个连接的epoll fd
func (c *Connect) GetEpFd() int {
	return c.epfd
}

//SetPoller .
func (c *Connect) SetPoller(poller iface.IPoller) {
	c.poller = poller
}

//SetWriteBuff .
func (c *Connect) SetWriteBuff(bytes []byte) {
	c.writeBuff = bytes
}

//GetWriteBuff .
func (c *Connect) GetWriteBuff() []byte {

	// 这里处理一下
	return c.writeBuff
}
