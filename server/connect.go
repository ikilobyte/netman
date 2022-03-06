package server

import (
	"bytes"
	"crypto/tls"
	"io"
	"net"
	"time"

	"github.com/ikilobyte/netman/common"

	"github.com/ikilobyte/netman/util"

	"github.com/ikilobyte/netman/iface"
	"golang.org/x/sys/unix"
)

//Connect TCP连接成功建立后，会抽象一个Connect
type Connect struct {
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
	readBuffer         *bytes.Buffer       // 未读取完整的一个数据包
	packDataLength     uint32              // 数据包体长度，如果这个值 == 0，那就是从头开始读取，没有未读取完整的数据
	temporaryMessage   iface.IMessage
}

//NewConnect 构造一个连接
func newConnect(id int, fd int, address net.Addr, options *Options) *Connect {
	connect := &Connect{
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
		readBuffer:         bytes.NewBuffer([]byte{}),
		packDataLength:     0,
		temporaryMessage:   nil,
	}

	// 执行回调
	if connect.hooks != nil {
		go connect.hooks.OnOpen(connect)
	}

	if options.TlsEnable {
		// 优先使用自定义的tlsConfig
		if options.TlsConfig != nil {
			connect.tlsLayer = tls.Server(connect, options.TlsConfig)
		} else {
			connect.tlsLayer = tls.Server(connect, &tls.Config{Certificates: []tls.Certificate{*options.TlsCertificate}})
		}
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

	// 移除事件监听
	_ = c.GetPoller().Remove(c.fd)

	// 从管理类中移除
	c.GetConnectMgr().Remove(c)

	// 关闭连接
	err := unix.Close(c.fd)

	// 重置为0
	c.packDataLength = 0

	// 重置
	c.readBuffer = nil

	// 关闭成功才执行
	if c.hooks != nil && err == nil {
		c.hooks.OnClose(c)
	}
	return err
}

// Read 读取数据
func (c *Connect) Read(bs []byte) (int, error) {

	n, err := unix.Read(c.fd, bs)

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
func (c *Connect) Write(dataPack []byte) (int, error) {

	// 当前连接是否为 EPOLLOUT 事件
	totalBytes := len(dataPack)
	if c.state == common.EPollOUT {
		c.writeQ.Push(dataPack)
		return totalBytes, nil
	}

	// 先尝试直接写数据，非阻塞情况下，可能无法全部写完整
	n, err := unix.Write(c.fd, dataPack)
	if err != nil {
		// FD 已断开
		if err == unix.EBADF || err == unix.EPIPE {
			_ = c.Close()
			return -1, err
		}
	}

	// 1、缓冲区满，无法写入，会返回	err = unix.EAGAIN
	// 2、客户端连接已断开，一般来说内核会延迟一会给出对应的err(unix.EPIPE, unix.EBADF)
	// 这种情况一般只有发送大量(MB)数据时才会出现
	if n != totalBytes && n > 0 {
		// 同时只能存在一个状态，要么可读，要么可写，禁止并行多个状态，可以把epoll理解为状态机
		// 注册可写事件，内核通知可写后，继续写入数据
		// 把剩下的保存到写入队列中
		c.SetState(common.EPollOUT)
		c.writeQ.Push(dataPack[n:])
		_ = c.poller.ModWrite(c.fd, c.id)

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

//GetPacker 获取packer
func (c *Connect) GetPacker() iface.IPacker {
	return c.packer
}

func (c *Connect) GetAddress() net.Addr {
	return c.Address
}

//Send 写数据
func (c *Connect) Send(msgID uint32, bytes []byte) (int, error) {

	// 1、封包
	dataPack, err := c.packer.Pack(msgID, bytes)
	if err != nil {
		return 0, err
	}

	// 2、发送
	if c.GetTLSEnable() {
		return c.tlsLayer.Write(dataPack)
	}
	return c.Write(dataPack)

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
func (c *Connect) GetWriteBuff() ([]byte, bool) {

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
func (c *Connect) SetState(state common.ConnectState) {
	c.state = state
}

//SetLastMessageTime 外部请勿调用
func (c *Connect) SetLastMessageTime(duration time.Time) {
	c.lastMessageTime = duration
}

func (c *Connect) GetTLSEnable() bool {
	return c.tlsEnable
}

func (c *Connect) GetHandshakeCompleted() bool {
	return c.handshakeCompleted
}

func (c *Connect) SetHandshakeCompleted() {
	c.handshakeCompleted = true
}

//GetCertificate 获取tls证书配置
func (c *Connect) GetCertificate() tls.Certificate {
	return *c.options.TlsCertificate
}

//GetTLSLayer 获取TLS层的对象
func (c *Connect) GetTLSLayer() *tls.Conn {
	return c.tlsLayer
}

//GetConnectMgr 获取connectMgr
func (c *Connect) GetConnectMgr() iface.IConnectManager {
	return c.GetPoller().GetConnectMgr()
}

//ProceedWrite 继续将未发送完毕的数据发送出去
func (c *Connect) ProceedWrite() error {

	// 1. 获取一个待发送的数据
	dataBuff, empty := c.GetWriteBuff()

	// 2. 队列中没有未发送完毕的数据，将当前连接改为可读事件
	if empty {

		// 更改为可读状态
		if err := c.GetPoller().ModRead(c.GetFd(), c.GetID()); err != nil {
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

//NonBlockingRead 非阻塞读取一个完整的数据包
func (c *Connect) NonBlockingRead() (iface.IMessage, error) {

	if c.packDataLength <= 0 {

		// 读取包头
		headBytes := make([]byte, c.packer.GetHeaderLength())
		n, err := c.readData(headBytes)

		// 连接断开
		if n == 0 && err == io.EOF {
			return nil, io.EOF
		}

		// 有错误，可能是 unix.EAGAIN 等错误
		if err != nil {
			// fd有异常
			if err == unix.EBADF || err == unix.EPIPE {
				return nil, io.EOF
			}
			return nil, err
		}

		// 包头数据读取有误
		if n != len(headBytes) {
			return nil, util.HeadBytesLengthFail
		}

		// 解包
		message, err := c.packer.UnPack(headBytes)
		if err != nil {
			return nil, err
		}

		// 包体长度为0
		if message.Len() <= 0 {
			return message, nil
		}

		// 设置长度数据
		c.packDataLength = uint32(message.Len())
		c.temporaryMessage = message
	}

	// 本次读取的最大长度
	readBytes := make([]byte, c.packDataLength-uint32(c.readBuffer.Len()))

	n, err := c.readData(readBytes)

	// 连接断开
	if n == 0 && err == io.EOF {
		return nil, err
	}

	// 读取数据有误
	if err != nil {

		// FD出现了异常
		if err == unix.EBADF || err == unix.EPIPE {
			return nil, io.EOF
		}

		// 保存到这里
		if n > 0 {
			c.readBuffer.Write(readBytes[:n])
		}
		return nil, err
	}

	// 将读取到的数据保存到这里
	c.readBuffer.Write(readBytes[:n])

	//fmt.Printf(
	//	"c.readData.n[%d] c.readData.err[%v] c.packDataLen[%d] c.readBuff.len[%d] readBytes.len[%d]\n",
	//	n,
	//	err,
	//	c.packDataLength,
	//	c.readBuffer.Len(),
	//	len(readBytes),
	//)

	// 数据包完整
	if c.readBuffer.Len() == int(c.packDataLength) {

		c.temporaryMessage.SetData(c.readBuffer.Bytes())

		// 重置，每个数据包都是一个互不影响的slice
		c.readBuffer = bytes.NewBuffer([]byte{})

		// 重置包体总长度
		c.packDataLength = 0

		return c.temporaryMessage, nil
	}

	return nil, nil
}

// 以下方法是为了实现TLS，实际并未实现

//GetLastMessageTime .
func (c *Connect) GetLastMessageTime() time.Time {
	return c.lastMessageTime
}

//GetPoller ..
func (c *Connect) GetPoller() iface.IPoller {
	return c.poller
}

//LocalAddr ..
func (c *Connect) LocalAddr() net.Addr {
	return nil
}

//RemoteAddr ..
func (c *Connect) RemoteAddr() net.Addr {
	return c.Address
}

//SetDeadline ..
func (c *Connect) SetDeadline(t time.Time) error {
	return nil
}

//SetReadDeadline ..
func (c *Connect) SetReadDeadline(t time.Time) error {
	return nil
}

//SetWriteDeadline ..
func (c *Connect) SetWriteDeadline(t time.Time) error {
	return nil
}

//readData 读取数据
func (c *Connect) readData(bs []byte) (int, error) {
	if c.GetTLSEnable() {
		return c.GetTLSLayer().Read(bs)
	}
	return c.Read(bs)
}
