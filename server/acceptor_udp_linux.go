// +build linux

package server

import (
	"fmt"
	"github.com/ikilobyte/netman/util"
	"log"
	"runtime"
	"syscall"
	"time"

	"golang.org/x/sys/unix"

	"github.com/ikilobyte/netman/eventloop"
	"github.com/ikilobyte/netman/iface"
)

type acceptorUdp struct {
	packer     iface.IPacker
	connectMgr iface.IConnectManager
	poller     *eventloop.Poller
	eventfd    int
	eventbuff  []byte
	connID     int
	options    *Options
}

func newAcceptorUdp(packer iface.IPacker, connectMgr iface.IConnectManager, options *Options) iface.IAcceptor {

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
	}
}

//Run 启动
func (a *acceptorUdp) Run(listenerFD int, loop iface.IEventLoop) error {

	for i := 0; i < runtime.NumCPU(); i++ {
		go func(idx int) {
			//udpSocket := newUdpSocket("0.0.0.0", 6565)
			//fmt.Println(udpSocket.fd)
			for {
				buffer := make([]byte, 1024)
				n, sockaddr, err := unix.Recvfrom(listenerFD, buffer, 0)
				if err != nil {
					fmt.Println("err", err)
					time.Sleep(time.Second)
					continue
				}
				fmt.Println(idx, "new connect", sockaddr, n, buffer[:n], listenerFD)
			}
		}(i)
	}

	time.Sleep(time.Hour)
	poller, err := eventloop.NewPoller(a.connectMgr)
	if err != nil {
		return err
	}

	// 添加eventfd
	if err := poller.AddRead(a.eventfd, a.IncrementID()); err != nil {
		return err
	}

	// 添加listener fd
	if err := poller.AddRead(listenerFD, a.IncrementID()); err != nil {
		return err
	}

	for {
		n, err := unix.EpollWait(poller.Epfd, poller.Events, -1)
		fmt.Println("epoll.wait", n, err)
		if err != nil {
			if err == unix.EAGAIN || err == unix.EINTR {
				continue
			}
			return err
		}

		for i := 0; i < n; i++ {
			event := poller.Events[i]
			fd := int(event.Fd)

			if fd == a.eventfd {
				_, _ = unix.Read(fd, a.eventbuff)
				a.Close()
				return nil
			}

			// 每次都是读数据的
			buffer := make([]byte, a.options.UDPPacketBufferLength)
			n, sockaddr, err := unix.Recvfrom(fd, buffer, 0)
			fmt.Println(buffer[:n], n)
			addr := util.SockaddrToUDPAddr(sockaddr)
			fmt.Println("addr.String() -> ", addr.String())
			if err != nil {
				if err == syscall.Errno(9) {
					a.Close()
					return nil
				}
				util.Logger.Errorf("acceptorUdp error: %v", err)
				continue
			}

			cfd, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, unix.IPPROTO_UDP)
			if err != nil {
				fmt.Println("cfd err", err)
			}
			_ = unix.Connect(cfd, sockaddr)
			fmt.Println("cfd", cfd)
			fmt.Println(unix.Write(cfd, []byte("hello world")))
			//fakeFD := a.IncrementID()
			//baseConnect := newBaseConnect(
			//	fakeFD,
			//	fakeFD,
			//	util.SockaddrToUDPAddr(sockaddr),
			//	a.options,
			//)
			//connect := newRouterProtocol(baseConnect) // 路由模式，也可以是自定义应用层协议
			//fmt.Println(connect)
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
