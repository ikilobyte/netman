// +build linux

package eventloop

import (
	"fmt"

	"golang.org/x/sys/unix"
)

type Poller struct {
	epfd int // eventpoll fd
}

//NewPoller 创建epoll
func NewPoller() (*Poller, error) {

	poller := new(Poller)
	fd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		return nil, err
	}
	poller.epfd = fd

	return poller, nil
}

func (p *Poller) Wait() {
	// TODO 待优化这里的逻辑
	fmt.Printf("poller.id %d Wait\n", p.epfd)
	select {}
}

//AddRead 添加读事件
func (p *Poller) AddRead(fd, pad int) error {
	return unix.EpollCtl(p.epfd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{
		Events: unix.EPOLLIN | unix.EPOLLPRI,
		Fd:     int32(fd),
		Pad:    int32(pad),
	})
}

//AddWrite 添加可写事件
func (p *Poller) AddWrite(fd, pad int) error {
	return unix.EpollCtl(p.epfd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{
		Events: unix.EPOLLOUT,
		Fd:     int32(fd),
		Pad:    int32(pad),
	})
}

//Remove 删除某个fd的事件
func (p *Poller) Remove(fd int) error {
	return unix.EpollCtl(p.epfd, unix.EPOLL_CTL_DEL, fd, nil)
}

//Close 关闭epoll
func (p *Poller) Close() error {
	return unix.Close(p.epfd)
}
