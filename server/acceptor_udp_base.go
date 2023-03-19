package server

import (
	"fmt"
	"github.com/ikilobyte/netman/iface"
	"github.com/ikilobyte/netman/util"
	"golang.org/x/sys/unix"
	"syscall"
)

//makeUdpConnect 将udp抽象为connect
func (a *acceptorUdp) makeUdpConnect(fd int, eventLoop iface.IEventLoop) (iface.IConnect, error) {

	buffer := make([]byte, a.options.UDPPacketBufferLength)
	n, sockaddr, err := unix.Recvfrom(fd, buffer, 0)
	headLen := int(a.packer.GetHeaderLength())
	address := util.SockaddrToUDPAddr(sockaddr).String()

	if err != nil {
		if err == syscall.Errno(9) {
			a.Close()
			return nil, err
		}
		return nil, fmt.Errorf("UDP acceptor from %s err: %v", address, err)
	}

	if n < int(a.packer.GetHeaderLength()) {
		return nil, fmt.Errorf("recv message No packet from %v", address)
	}

	message, err := a.packer.UnPack(buffer[:headLen])

	if err != nil {
		return nil, fmt.Errorf("unpack message err %v", err)
	}

	if n-headLen != message.Len() {
		return nil, fmt.Errorf("not a complete data packet")
	}

	// 创建一个socket，用于绑定
	udpFD, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, unix.IPPROTO_UDP)
	if err != nil {
		return nil, fmt.Errorf("create udp socket err %v", err)
	}

	// reuseport
	if err := unix.SetsockoptInt(udpFD, unix.SOL_SOCKET, unix.SO_REUSEPORT, 1); err != nil {
		return nil, fmt.Errorf("set option SO_REUSEPORT err %v", err)
	}

	// reuseaddr
	if err := unix.SetsockoptInt(udpFD, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1); err != nil {
		return nil, fmt.Errorf("set option SO_REUSEADDR err %v", err)
	}

	if err := unix.Bind(udpFD, a.server.socket.sockAddr); err != nil {
		return nil, fmt.Errorf("udp bind addr err %v", err)
	}

	if err := unix.Connect(udpFD, sockaddr); err != nil {
		return nil, fmt.Errorf("udp connect err %v", err)
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
	if err := eventLoop.AddRead(connect); err != nil {
		_ = connect.Close()
		return nil, err
	}

	// 添加到全局管理中
	a.connectMgr.Add(connect)

	// 发送一次出去即可
	message.SetData(buffer[headLen : headLen+message.Len()])
	context := util.NewContext(util.NewRequest(connect, message, a.connectMgr))
	a.server.emitCh <- context

	return connect, nil
}
