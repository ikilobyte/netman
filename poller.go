package netman

import (
	"fmt"

	"golang.org/x/sys/unix"
)

type poller struct {
	epfd int // eventpoll fd
}

//newPoller 创建epoll
func newPoller() (*poller, error) {

	poller := new(poller)
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
			fmt.Println(ev.Fd)

			buff := make([]byte, 512)
			for {
				n, err := unix.Read(int(ev.Fd), buff)
				if err != nil {
					if err == unix.EAGAIN {
						break
					}
					fmt.Println("recv.err", err)
					break
				}
				fmt.Println("recv", string(buff[:n]))
			}
		}
		fmt.Println("n", n)
	}
}

//AddRead 添加读事件
func (p *poller) AddRead(fd int) error {
	return unix.EpollCtl(p.epfd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{
		Events: unix.EPOLLIN | unix.EPOLLPRI,
		Fd:     int32(fd),
	})
}

//AddWrite 添加可写事件
func (p *poller) AddWrite(fd int) error {
	return unix.EpollCtl(p.epfd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{
		Events: unix.EPOLLOUT,
		Fd:     int32(fd),
	})
}

//Remove 删除某个fd的事件
func (p *poller) Remove(fd int) error {
	return unix.EpollCtl(p.epfd, unix.EPOLL_CTL_DEL, fd, nil)
}

func (p *poller) close() {
	unix.Close(p.epfd)
}
