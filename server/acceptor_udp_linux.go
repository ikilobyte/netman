// +build linux

package server

import (
	"github.com/ikilobyte/netman/eventloop"
	"github.com/ikilobyte/netman/iface"
	"github.com/ikilobyte/netman/util"
	"golang.org/x/sys/unix"
	"log"
	"syscall"
)

type acceptorUdp struct {
	packer     iface.IPacker
	connectMgr iface.IConnectManager
	poller     *eventloop.Poller
	eventfd    int
	eventbuff  []byte
	connID     int
	options    *Options
	server     *Server
}

func newAcceptorUdp(packer iface.IPacker, connectMgr iface.IConnectManager, options *Options, server *Server) iface.IAcceptor {

	eventfd, err := unix.Eventfd(0, unix.EPOLL_CLOEXEC)
	if err != nil {
		log.Panicln(err)
	}

	poller, err := eventloop.NewPoller(connectMgr)
	if err != nil {
		log.Panicln(err)
	}

	return &acceptorUdp{
		packer:     packer,
		connectMgr: connectMgr,
		poller:     poller,
		eventfd:    eventfd,
		eventbuff:  []byte{0, 0, 0, 0, 0, 0, 0, 1},
		connID:     -1,
		options:    options,
		server:     server,
	}
}

//Run 启动，只用于接收新的"连接"
// UDP 没有连接的概念，但可以参考TCP，手动创建一个fd，结合epoll，达到多路复用
func (a *acceptorUdp) Run(listenerFD int, loop iface.IEventLoop) error {

	// 添加eventfd，用于server退出
	if err := a.poller.AddRead(a.eventfd, a.IncrementID()); err != nil {
		return err
	}

	// 添加listener fd
	// 虽然udp没有accept的概念，但是可以使用listener的方式创造一个连接
	if err := a.poller.AddRead(listenerFD, a.IncrementID()); err != nil {
		return err
	}

	headLen := a.packer.GetHeaderLength()

	for {
		n, err := unix.EpollWait(a.poller.Epfd, a.poller.Events, -1)
		if err != nil {
			if err == unix.EAGAIN || err == unix.EINTR {
				continue
			}
			return err
		}

		for i := 0; i < n; i++ {
			event := a.poller.Events[i]
			fd := int(event.Fd)

			// close
			if fd == a.eventfd {
				_, _ = unix.Read(fd, a.eventbuff)
				a.Close()
				return nil
			}

			buffer := make([]byte, a.options.UDPPacketBufferLength)
			n, sockaddr, err := unix.Recvfrom(fd, buffer, 0)

			if err != nil {
				if err == syscall.Errno(9) {
					a.Close()
					return nil
				}
				util.Logger.Errorf("UDP acceptor err: %v", err)
				continue
			}

			if n < int(a.packer.GetHeaderLength()) {
				util.Logger.Errorf("recv message No packet from %v", util.SockaddrToUDPAddr(sockaddr).String())
				continue
			}

			message, err := a.packer.UnPack(buffer[:headLen])

			if err != nil {
				util.Logger.Errorf("unpack message err %v", err)
				continue
			}

			if n-int(headLen) != message.Len() {
				util.Logger.Errorf("Not a complete data packet")
				continue
			}

			// 创建一个socket，用于绑定
			udpFD, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, unix.IPPROTO_UDP)
			if err != nil {
				util.Logger.Errorf("create udp socket err %v", err)
				continue
			}

			// reuseport
			if err := unix.SetsockoptInt(udpFD, unix.SOL_SOCKET, unix.SO_REUSEPORT, 1); err != nil {
				util.Logger.Errorf("set option SO_REUSEPORT err %v", err)
				continue
			}

			// reuseaddr
			if err := unix.SetsockoptInt(udpFD, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1); err != nil {
				util.Logger.Errorf("set option SO_REUSEADDR err %v", err)
				continue
			}

			if err := unix.Bind(udpFD, a.server.socket.sockArrd); err != nil {
				util.Logger.Errorf("udp bind addr err %v", err)
				continue
			}

			if err := unix.Connect(udpFD, sockaddr); err != nil {
				util.Logger.Errorf("udp connect err %v", err)
				continue
			}

			// 封装成connect，方便管理
			baseConnect := newBaseConnect(
				a.IncrementID(),
				udpFD,
				util.SockaddrToUDPAddr(sockaddr),
				a.options,
			)

			connect := newRouterProtocol(baseConnect) // 路由模式，也可以是自定义应用层协议

			// 添加到事件循环
			if err := loop.AddRead(connect); err != nil {
				_ = connect.Close()
				continue
			}

			// 添加到全局管理中
			a.connectMgr.Add(connect)

			// 发送一次出去即可
			message.SetData(buffer[headLen:message.Len()])
			a.server.emitCh <- util.NewContext(util.NewRequest(connect, message, a.connectMgr))
		}
	}
}

func (a *acceptorUdp) IncrementID() int {
	a.connID += 1
	return a.connID
}

func (a *acceptorUdp) Close() {
	_ = a.poller.Remove(a.eventfd)
	_ = unix.Close(a.eventfd)
	_ = a.poller.Close()
}

func (a *acceptorUdp) Exit() {
	_, _ = unix.Write(a.eventfd, a.eventbuff)
}
