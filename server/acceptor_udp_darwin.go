// +build darwin

package server

import (
	"github.com/ikilobyte/netman/eventloop"
	"github.com/ikilobyte/netman/iface"
	"github.com/ikilobyte/netman/util"
	"golang.org/x/sys/unix"
	"log"
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

	poller, err := eventloop.NewPoller(connectMgr)
	if err != nil {
		log.Panicln(err)
	}

	return &acceptorUdp{
		packer:     packer,
		connectMgr: connectMgr,
		poller:     poller,
		eventbuff:  []byte{},
		eventfd:    0,
		connID:     -1,
		options:    options,
		server:     server,
	}
}

//Run 启动，只用于接收新的"连接"
// UDP 没有连接的概念，但可以参考TCP，手动创建一个fd，结合epoll，达到多路复用
func (a *acceptorUdp) Run(listenerFD int, loop iface.IEventLoop) error {

	// 添加event
	if _, err := unix.Kevent(a.poller.Epfd, []unix.Kevent_t{
		{Ident: 0, Filter: unix.EVFILT_USER, Flags: unix.EV_ADD | unix.EV_CLEAR},
	}, nil, nil); err != nil {
		return err
	}

	// 添加listener fd
	// 虽然udp没有accept的概念，但是可以使用listener的方式创造一个连接
	if err := a.poller.AddRead(listenerFD, a.IncrementID()); err != nil {
		return err
	}

	for {
		n, err := unix.Kevent(a.poller.Epfd, nil, a.poller.Events, nil)
		if err != nil {
			if err == unix.EAGAIN || err == unix.EINTR {
				continue
			}
			return err
		}

		for i := 0; i < n; i++ {
			event := a.poller.Events[i]
			fd := int(event.Ident)

			// close
			if fd == a.eventfd {
				a.Close()
				return nil
			}

			// 创建一个udp client
			if _, err := a.makeUdpConnect(fd, loop); err != nil {
				util.Logger.Errorln("make udp connect err %v", err)
			}
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
	_, _ = unix.Kevent(a.poller.Epfd, []unix.Kevent_t{{
		Ident:  0,
		Filter: unix.EVFILT_USER,
		Fflags: unix.NOTE_TRIGGER,
	}}, nil, nil)
}
