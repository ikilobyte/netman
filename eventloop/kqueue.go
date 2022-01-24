// +build darwin freebsd dragonfly

package eventloop

import (
	"io"

	"github.com/ikilobyte/netman/iface"
	"github.com/ikilobyte/netman/util"
	"golang.org/x/sys/unix"
)

type Poller struct {
	Epfd       int                   // eventpoll fd
	Events     []unix.Kevent_t       //
	ConnectMgr iface.IConnectManager //
}

//NewPoller 创建kqueue
func NewPoller(connectMgr iface.IConnectManager) (*Poller, error) {

	fd, err := unix.Kqueue()
	if err != nil {
		return nil, err
	}

	return &Poller{
		Epfd:       fd,
		Events:     make([]unix.Kevent_t, 128),
		ConnectMgr: connectMgr,
	}, nil
}

func (p *Poller) AddRead(fd int, connID int) error {
	_, err := unix.Kevent(p.Epfd, []unix.Kevent_t{
		{
			Ident:  uint64(fd),
			Filter: unix.EVFILT_READ,
			Flags:  unix.EV_ADD,
			Fflags: 0,
			Data:   int64(connID),
			Udata:  nil,
		},
	}, nil, nil)
	return err
}

func (p *Poller) AddWrite(fd, connID int) error {
	_, err := unix.Kevent(p.Epfd, []unix.Kevent_t{
		{
			Ident:  uint64(fd),
			Filter: unix.EVFILT_WRITE,
			Flags:  unix.EV_ADD,
			Fflags: 0,
			Data:   int64(connID),
			Udata:  nil,
		},
	}, nil, nil)
	return err
}

//Wait 这里处理的是socket的读
func (p *Poller) Wait(emitCh chan<- iface.IRequest) {

	for {

		n, err := unix.Kevent(p.Epfd, nil, p.Events, nil)
		if err != nil {
			if err == unix.EINTR || err == unix.EAGAIN {
				continue
			}

			util.Logger.WithField("epfd", p.Epfd).WithField("error", err).Error("kqueue wait error")

			// 断开这个epoll管理的所有连接
			p.ConnectMgr.ClearByEpFd(p.Epfd)

			return
		}

		// 处理连接
		for i := 0; i < n; i++ {
			var (
				event  = p.Events[i]
				connFd = int(event.Ident)
				connID = int(event.Data) // TODO bug
				conn   iface.IConnect
			)

			// 1、通过connID获取conn实例
			if conn = p.ConnectMgr.Get(connID); conn == nil {
				// 断开连接
				_ = unix.Close(connFd)
				_ = p.Remove(connFd)
				continue
			}

			// 2、读取一个完整的包
			message, err := conn.GetPacker().ReadFull(connFd)
			if err != nil {

				// 这两种情况可以直接断开连接
				if err == io.EOF || err == util.HeadBytesLengthFail {

					// 断开连接操作
					_ = conn.Close()
					_ = p.Remove(connFd)
					p.ConnectMgr.Remove(conn)
				}
				continue
			}

			// 3、将消息传递出去，交给worker处理
			if message.Len() <= 0 {
				continue
			}

			emitCh <- util.NewRequest(conn, message)
		}
	}
}

func (p *Poller) Remove(fd int) error {
	return nil
}

func (p *Poller) Close() error {
	return unix.Close(p.Epfd)
}
