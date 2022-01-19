// +build linux

package eventloop

import (
	"fmt"

	"github.com/ikilobyte/netman/iface"
	"github.com/ikilobyte/netman/util"

	"golang.org/x/sys/unix"
)

type Poller struct {
	epfd       int // eventpoll fd
	events     []unix.EpollEvent
	connectMgr iface.IConnectManager
}

//NewPoller 创建epoll
func NewPoller(connectMgr iface.IConnectManager) (*Poller, error) {

	poller := &Poller{
		epfd:       0,
		events:     make([]unix.EpollEvent, 128),
		connectMgr: connectMgr,
	}
	fd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		return nil, err
	}
	poller.epfd = fd

	return poller, nil
}

func (p *Poller) Wait() {

	for {
		// n有三种情况，-1，0，> 0
		n, err := unix.EpollWait(p.epfd, p.events, -1)
		if err != nil {
			if err == unix.EAGAIN {
				continue
			}

			if err == unix.EINTR {
				continue
			}

			fmt.Printf("epoll_wait err %s\n", err)
			break
		}

		for i := 0; i < n; i++ {

			var (
				event  = p.events[i]
				connFd = int(event.Fd)
				connID = int(event.Pad)
				//headBytes = make([]byte, 8)
			)

			// 1、通过connID获取conn实例
			conn := p.connectMgr.Get(connID)

			// 获取不到
			if conn == nil {
				// 断开连接
				unix.Close(connFd)

				// 删除事件
				p.Remove(connFd)
				continue
			}

			message, err := conn.GetPacker().ReadFull(connFd)
			if err != nil {

				fmt.Println("ReadFull err", err)
				// 连接断开
				if err == util.ConnectClosed {
					fmt.Println("连接断开")
				}

				conn.Close()
				p.Remove(connFd)
				p.connectMgr.Remove(conn)
				continue
			}

			//echo sprintf('data.size %d msgID %d',strlen($string),$msgId) . "\n";
			fmt.Printf("data.size %d msgID %d\n\n\n", message.Len(), message.ID())
		}
	}
}

//AddRead 添加读事件
func (p *Poller) AddRead(fd, connID int) error {
	return unix.EpollCtl(p.epfd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{
		Events: unix.EPOLLIN | unix.EPOLLPRI,
		Fd:     int32(fd),
		Pad:    int32(connID),
	})
}

//AddWrite 添加可写事件
func (p *Poller) AddWrite(fd, connID int) error {
	return unix.EpollCtl(p.epfd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{
		Events: unix.EPOLLOUT,
		Fd:     int32(fd),
		Pad:    int32(connID),
	})
}

//Remove 删除某个fd的事件
func (p *Poller) Remove(fd int) error {
	return unix.EpollCtl(p.epfd, unix.EPOLL_CTL_DEL, fd, nil)
}

//Close 关闭FD
func (p *Poller) Close() error {
	return unix.Close(p.epfd)
}
