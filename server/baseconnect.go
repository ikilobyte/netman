package server

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/ikilobyte/netman/common"

	"github.com/ikilobyte/netman/util"

	"github.com/ikilobyte/netman/iface"
	"golang.org/x/sys/unix"
)

type BaseConnect struct {
	id                 int                 // 自定义生成的ID
	fd                 int                 // 系统分配的fd
	epfd               int                 // 管理这个连接的epoll
	packer             iface.IPacker       // 封包解包实现，可以自行实现
	Address            net.Addr            //
	hooks              iface.IHooks        //
	writeBuff          []byte              // 待发送的数据缓冲，如果这个变为空，那就表示这一次的全部发送完毕了！
	poller             iface.IPoller       //
	writeQ             *util.Queue         //
	state              common.ConnectState // 当前状态，0 离线，1 在线，2 epoll状态是可写，3 epoll状态是可读
	lastMessageTime    time.Time           // 最后一次发送消息的时间，用于心跳检测
	tlsEnable          bool                // 是否开启了tls
	handshakeCompleted bool                // tls握手是否完成
	options            *Options            // 可选项配置
	tlsLayer           *tls.Conn           // TLS层
	tlsRawSize         int                 // tls原始字节数据，对应*tls.Conn.rawInput中是否还有数据未读
	tlsWritePacketSize int                 // 发送数据包的长度
}

func newBaseConnect(id int, fd int, address net.Addr, options *Options) *BaseConnect {
	connect := &BaseConnect{
		id:                 id,
		fd:                 fd,
		packer:             options.Packer,
		Address:            address,
		hooks:              options.Hooks,
		writeQ:             util.NewQueue(), // 待发送的数据队列
		state:              common.OnLine,   // 状态
		lastMessageTime:    time.Now(),      // 初始化
		tlsEnable:          options.TlsEnable,
		handshakeCompleted: false,
		options:            options,
		tlsLayer:           nil,
		tlsRawSize:         0,
	}

	// TLS相关配置
	if connect.options.TlsEnable {
		if connect.options.TlsConfig != nil {
			connect.tlsLayer = tls.Server(connect, connect.options.TlsConfig)
		} else {
			connect.tlsLayer = tls.Server(connect, &tls.Config{Certificates: []tls.Certificate{*connect.options.TlsCertificate}})
		}
	}

	// 执行onopen事件
	if connect.hooks != nil {
		go connect.hooks.OnOpen(connect)
	}

	return connect
}

//GetID 获取连接ID
func (c *BaseConnect) GetID() int {
	return c.id
}

//GetFd 获取系统分配的fd
func (c *BaseConnect) GetFd() int {
	return c.fd
}

// Read 读取数据
func (c *BaseConnect) Read(bs []byte) (int, error) {

	n, err := unix.Read(c.fd, bs)

	// 已完成了TLS握手
	if c.handshakeCompleted {
		if n >= 0 {
			c.tlsRawSize += n
		}
	}

	// 内核返回错误，为了兼容TLS库，不能返回-1
	if n < 0 {
		return 0, err
	}

	// 连接已断开，读取的字节是0
	if n == 0 {
		return 0, io.EOF
	}

	if err != nil {
		return n, err
	}

	return n, nil
}

//Write ..只是为了实现tls，请勿调用此方法，应该调用Send方法
func (c *BaseConnect) Write(dataPack []byte) (int, error) {

	// 当前连接是否为 EPOLLOUT 事件
	totalBytes := len(dataPack)
	if c.state == common.EPollOUT {
		c.writeQ.Push(dataPack)
		return totalBytes, nil
	}

	// 当前是TLS模式，且是非阻塞模式
	if c.GetHandshakeCompleted() {
		if err := unix.SetNonblock(c.fd, false); err != nil {
			return -1, io.EOF
		}
	}

	n, err := unix.Write(c.fd, dataPack)

	if err != nil {
		// FD 已断开
		if err == unix.EBADF || err == unix.EPIPE {
			//_ = c.Close()
			return -1, err
		}
	}

	// 设置为非阻塞模式
	if c.GetHandshakeCompleted() && n == totalBytes {
		_ = unix.SetNonblock(c.fd, true)
	}

	// 1、缓冲区满，无法写入，可能会返回	err = unix.EAGAIN ，但是也可能不会返回任何错误
	// 2、客户端连接已断开，一般来说内核会延迟一会给出对应的err(unix.EPIPE, unix.EBADF)
	// 这种情况一般只有发送大量(MB)数据时才会出现
	if n != totalBytes && n > 0 {
		// 同时只能存在一个状态，要么可读，要么可写，禁止并行多个状态，可以把epoll理解为状态机
		// 注册可写事件，内核通知可写后，继续写入数据
		// 把剩下的保存到写入队列中
		c.SetState(common.EPollOUT)
		c.writeQ.Push(dataPack[n:])
		_ = c.poller.ModWrite(c.fd, c.id)

		fmt.Println("????!等待下次可写？", err)
		return totalBytes, nil
	}

	// 一个字节都未发送出去，把打包好的数据放入到写入队列中
	if n < 0 {
		c.SetState(common.EPollOUT)
		c.writeQ.Push(dataPack)
		_ = c.poller.ModWrite(c.fd, c.id)

		return totalBytes, nil
	}
	return n, err
}

//Text ..
func (c *BaseConnect) Text(bytes []byte) (int, error) {
	return 0, nil
}

//Binary ..
func (c *BaseConnect) Binary(bytes []byte) (int, error) {
	return 0, nil
}

//GetPacker 获取packer
func (c *BaseConnect) GetPacker() iface.IPacker {
	return c.packer
}

func (c *BaseConnect) GetAddress() net.Addr {
	return c.Address
}

//SetEpFd 设置这个连接属于哪个epoll
func (c *BaseConnect) SetEpFd(epfd int) {
	c.epfd = epfd
}

//GetEpFd 获取这个连接的epoll fd
func (c *BaseConnect) GetEpFd() int {
	return c.epfd
}

//SetPoller .
func (c *BaseConnect) SetPoller(poller iface.IPoller) {
	c.poller = poller
}

//SetWriteBuff .
func (c *BaseConnect) SetWriteBuff(bytes []byte) {
	c.writeBuff = bytes
}

//GetWriteBuff .
func (c *BaseConnect) GetWriteBuff() ([]byte, bool) {

	// 从队列取出的数据还有未发送完毕的，需要发送剩余的字节
	if len(c.writeBuff) >= 1 {
		return c.writeBuff, false
	}

	// 从队列中取出一个，并暂存在这里，因为可能一次也不能完全发送出去
	empty := false
	dataPack := c.writeQ.Pop()

	// 队列中的数据全部发送完毕
	if dataPack == nil {
		c.writeBuff = []byte{}
		empty = true
	} else {
		c.writeBuff = dataPack.([]byte)
	}

	// 这里处理一下
	return c.writeBuff, empty
}

//SetState state取值范围 0 离线，1 在线，2 epoll状态是可写，3 epoll状态是可读
func (c *BaseConnect) SetState(state common.ConnectState) {
	c.state = state
}

//SetLastMessageTime .
func (c *BaseConnect) SetLastMessageTime(duration time.Time) {
	c.lastMessageTime = duration
}

func (c *BaseConnect) GetTLSEnable() bool {
	return c.tlsEnable
}

func (c *BaseConnect) GetHandshakeCompleted() bool {
	return c.handshakeCompleted
}

func (c *BaseConnect) SetHandshakeCompleted() {
	c.handshakeCompleted = true
}

//GetCertificate 获取tls证书配置
func (c *BaseConnect) GetCertificate() tls.Certificate {
	return *c.options.TlsCertificate
}

//GetTLSLayer 获取TLS层的对象
func (c *BaseConnect) GetTLSLayer() *tls.Conn {
	return c.tlsLayer
}

//GetConnectMgr 获取connectMgr
func (c *BaseConnect) GetConnectMgr() iface.IConnectManager {
	return c.GetPoller().GetConnectMgr()
}

//ProceedWrite 继续将未发送完毕的数据发送出去
func (c *BaseConnect) ProceedWrite() error {

	// 1. 获取一个待发送的数据
	dataBuff, empty := c.GetWriteBuff()

	// 2. 队列中没有未发送完毕的数据，将当前连接改为可读事件
	if empty {

		// 更改为可读状态
		if err := c.GetPoller().ModRead(c.fd, c.id); err != nil {
			return err
		}

		// 同步状态
		c.SetState(common.EPollIN)

		return nil
	}

	// 3. 发送
	n, err := unix.Write(c.GetFd(), dataBuff)

	// fmt.Printf("dataBuff %d empty %v 已发送[%d] 剩余[%d]\n", len(dataBuff), empty, n, len(dataBuff)-n)
	if err != nil {
		return err
	}

	// 设置 writeBuff
	c.SetWriteBuff(dataBuff[n:])

	return nil
}

//Close 会被重写，不会执行到这里
func (c *BaseConnect) Close() error {
	return nil
}

//Send 会被重写，不会执行到这里
func (c *BaseConnect) Send(msgID uint32, bs []byte) (int, error) {
	return 0, nil
}

// 以下方法是为了实现TLS，实际并未实现

//GetLastMessageTime .
func (c *BaseConnect) GetLastMessageTime() time.Time {
	return c.lastMessageTime
}

//GetPoller ..
func (c *BaseConnect) GetPoller() iface.IPoller {
	return c.poller
}

//LocalAddr ..
func (c *BaseConnect) LocalAddr() net.Addr {
	return nil
}

//RemoteAddr ..
func (c *BaseConnect) RemoteAddr() net.Addr {
	return c.Address
}

//SetDeadline ..
func (c *BaseConnect) SetDeadline(t time.Time) error {
	return nil
}

//SetReadDeadline ..
func (c *BaseConnect) SetReadDeadline(t time.Time) error {
	return nil
}

//SetWriteDeadline ..
func (c *BaseConnect) SetWriteDeadline(t time.Time) error {
	return nil
}

//readData 读取数据
func (c *BaseConnect) readData(bs []byte) (int, error) {
	if c.GetTLSEnable() {
		return c.GetTLSLayer().Read(bs)
	}
	return c.Read(bs)
}
