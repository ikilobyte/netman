// +build linux

package poller

import (
	"fmt"

	"github.com/ikilobyte/netman/iface"
	"golang.org/x/sys/unix"
)

type poller struct {
	epfd int // eventpoll fd
	svr  iface.IServer
}

//newPoller 创建epoll
func newPoller(svr iface.IServer) (*poller, error) {

	poller := new(poller)
	poller.svr = svr
	fd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		return nil, err
	}
	poller.epfd = fd

	return poller, nil
}

func (p *poller) Wait() {

	events := make([]unix.EpollEvent, 100)
	for {

		n, err := unix.EpollWait(p.epfd, events, -1)
		if err != nil {

			if err == unix.EAGAIN {
				continue
			}

			if err == unix.EINTR {
				continue
			}
			fmt.Println("qwrqwr", err)
			p.close()
			break
		}

		for i := 0; i < n; i++ {
			ev := events[i]

			dataBuff := make([]byte, 512)
			size := 0
			for {

				// 从这里解包，然后将message返回出去
				n, err = unix.Read(int(ev.Fd), dataBuff)

				// 断开了连接
				if n == 0 {

					// 断开连接
					unix.Close(int(ev.Fd))

					// 从event loop删除
					p.Remove(int(ev.Fd))

					// 从mgr中删除
					p.svr.GetConnMgr().Remove(int(ev.Pad))

					break
				}

				if err != nil {
					if err == unix.EAGAIN {
						break
					}
					fmt.Println("recv.err", err)
					break
				}
				size += n
			}

			if size >= 1 {
				// 分发出去给worker处理业务逻辑
				p.svr.Emit(dataBuff[:size])
			}
		}
	}
}

//AddRead 添加读事件
func (p *poller) AddRead(fd, pad int) error {
	return unix.EpollCtl(p.epfd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{
		Events: unix.EPOLLIN | unix.EPOLLPRI,
		Fd:     int32(fd),
		Pad:    int32(pad),
	})
}

//AddWrite 添加可写事件
func (p *poller) AddWrite(fd, pad int) error {
	return unix.EpollCtl(p.epfd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{
		Events: unix.EPOLLOUT,
		Fd:     int32(fd),
		Pad:    int32(pad),
	})
}

//Remove 删除某个fd的事件
func (p *poller) Remove(fd int) error {
	return unix.EpollCtl(p.epfd, unix.EPOLL_CTL_DEL, fd, nil)
}

func (p *poller) close() {
	unix.Close(p.epfd)
}
