package server

import (
	"log"
	"syscall"

	"github.com/ikilobyte/netman/util"

	"golang.org/x/sys/unix"

	"github.com/ikilobyte/netman/eventloop"
	"github.com/ikilobyte/netman/iface"
)

//acceptor 统一处理用来处理新连接
type acceptor struct {
	packer     iface.IPacker
	connectMgr iface.IConnectManager
	eventfd    int
	eventbuff  []byte
	connID     int
}

func newAcceptor(packer iface.IPacker, connectMgr iface.IConnectManager) *acceptor {

	efd, err := unix.Eventfd(0, unix.EPOLL_CLOEXEC)
	if err != nil {
		log.Panicln(err)
	}

	return &acceptor{
		packer:     packer,
		connectMgr: connectMgr,
		eventfd:    efd,
		eventbuff:  []byte{0, 0, 0, 0, 0, 0, 0, 1},
		connID:     -1,
	}
}

//Start 启动
func (a *acceptor) Start(listenerFd int, loop iface.IEventLoop) error {

	poller, err := eventloop.NewPoller(a.connectMgr)
	if err != nil {
		return err
	}

	// 添加eventfd
	if err := poller.AddRead(a.eventfd, 0); err != nil {
		return err
	}

	// 添加listener fd
	if err := poller.AddRead(listenerFd, 1); err != nil {
		return err
	}

	for {
		n, err := unix.EpollWait(poller.Epfd, poller.Events, -1)
		if err != nil {
			if err == unix.EAGAIN || err == unix.EINTR {
				continue
			}
			return err
		}

		for i := 0; i < n; i++ {
			event := poller.Events[i]
			eventFd := int(event.Fd)

			if eventFd == a.eventfd {
				_, _ = unix.Read(eventFd, a.eventbuff)
				a.Close(poller)
				return nil
			}

			connFd, sa, err := unix.Accept(eventFd)
			if err != nil {
				if err == syscall.Errno(9) {
					a.Close(poller)
					return nil
				}
				util.Logger.Errorf("acceptor error: %v", err)
				continue
			}

			// 设置非阻塞
			if err := unix.SetNonblock(connFd, true); err != nil {
				_ = unix.Close(connFd)
				continue
			}

			// 设置不延迟
			if err := unix.SetsockoptInt(connFd, syscall.IPPROTO_TCP, syscall.TCP_NODELAY, 1); err != nil {
				_ = unix.Close(connFd)
				continue
			}

			connect := newConnect(
				a.IncrementID(),
				connFd,
				util.SockaddrToTCPOrUnixAddr(sa),
				a.packer,
			)

			// 添加事件循环
			if err := loop.AddRead(connect); err != nil {
				_ = connect.Close()
				continue
			}

			// 添加到这里
			a.connectMgr.Add(connect)
		}
	}
}

func (a *acceptor) IncrementID() int {
	a.connID += 1
	return a.connID
}

func (a *acceptor) Close(poller *eventloop.Poller) {
	_ = poller.Remove(a.eventfd)
	_ = unix.Close(a.eventfd)
	_ = poller.Close()
}
