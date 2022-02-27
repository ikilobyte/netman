// +build linux

package eventloop

import (
	"io"

	"github.com/ikilobyte/netman/common"
	"github.com/ikilobyte/netman/util"

	"github.com/ikilobyte/netman/iface"
	"golang.org/x/sys/unix"
)

type Poller struct {
	Epfd       int                   // eventpoll fd
	Events     []unix.EpollEvent     //
	ConnectMgr iface.IConnectManager //
}

//NewPoller 创建epoll
func NewPoller(connectMgr iface.IConnectManager) (*Poller, error) {

	fd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		return nil, err
	}

	return &Poller{
		Epfd:       fd,
		Events:     make([]unix.EpollEvent, 128),
		ConnectMgr: connectMgr,
	}, nil
}

//Wait 等待消息到达，通过通道传递出去
func (p *Poller) Wait(emitCh chan<- iface.IRequest) {

	for {
		// n有三种情况，-1，0，> 0
		n, err := unix.EpollWait(p.Epfd, p.Events, -1)
		if err != nil {
			if err == unix.EAGAIN || err == unix.EINTR {
				continue
			}

			util.Logger.WithField("epfd", p.Epfd).WithField("error", err).Error("epoll_wait error")
			// 断开这个epoll管理的所有连接
			p.ConnectMgr.ClearByEpFd(p.Epfd)
			return
		}

		for i := 0; i < n; i++ {

			var (
				event  = p.Events[i]
				connFd = int(event.Fd)
				conn   iface.IConnect
			)

			// 1、通过connID获取conn实例
			if conn = p.ConnectMgr.Get(connFd); conn == nil {
				// 断开连接
				_ = unix.Close(connFd)
				_ = p.Remove(connFd)
				continue
			}

			// 可写事件
			if event.Events&unix.EPOLLOUT == unix.EPOLLOUT {

				// 继续写
				if err := p.ProceedWrite(conn); err != nil {
					_ = conn.Close()     // 断开连接
					_ = p.Remove(connFd) // 删除事件订阅
					p.ConnectMgr.Remove(conn)
					util.Logger.Errorf("epoll do write error %v", err)
					continue
				}
				continue
			}

			// 1、判断是否开启tls
			if conn.GetTLSEnable() && conn.GetHandshakeCompleted() == false {

				tlsConnect := conn.GetTLSConnect()
				if err := tlsConnect.Handshake(); err != nil {
					p.ClearByConn(conn)
					util.Logger.Errorf("tls handshake error %v", err)
					continue
				}
				// 1、设置状态
				conn.SetHandshakeCompleted()

				// 2、设置为非阻塞
				// TODO 非阻塞模式
				//if err := unix.SetNonblock(connFd, true); err != nil {
				//	p.ClearByConn(conn)
				//	continue
				//}
			}

			// 2、读取一个完整的包
			message, err := conn.GetPacker().ReadFull(conn)
			if err != nil {
				switch err {
				case io.EOF, util.HeadBytesLengthFail, util.BodyLenExceedLimit:
					// 断开连接操作
					p.ClearByConn(conn)
				default:
					continue
				}
			}

			// 3、将消息传递出去，交给worker处理
			if message == nil || message.Len() <= 0 {
				continue
			}

			emitCh <- util.NewRequest(conn, message)
		}
	}
}

//AddRead 添加读事件
func (p *Poller) AddRead(fd, connID int) error {
	return unix.EpollCtl(p.Epfd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{
		Events: unix.EPOLLIN | unix.EPOLLPRI,
		Fd:     int32(fd),
		Pad:    int32(connID),
	})
}

//AddWrite 添加可写事件
func (p *Poller) AddWrite(fd, connID int) error {
	return unix.EpollCtl(p.Epfd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{
		Events: unix.EPOLLOUT,
		Fd:     int32(fd),
		Pad:    int32(connID),
	})
}

//ModWrite .
func (p *Poller) ModWrite(fd, connID int) error {
	return unix.EpollCtl(p.Epfd, unix.EPOLL_CTL_MOD, fd, &unix.EpollEvent{
		Events: unix.EPOLLOUT,
		Fd:     int32(fd),
		Pad:    int32(connID),
	})
}

//ModRead .
func (p *Poller) ModRead(fd, connID int) error {
	return unix.EpollCtl(p.Epfd, unix.EPOLL_CTL_MOD, fd, &unix.EpollEvent{
		Events: unix.EPOLLIN | unix.EPOLLPRI,
		Fd:     int32(fd),
		Pad:    int32(connID),
	})
}

//Remove 删除某个fd的事件
func (p *Poller) Remove(fd int) error {
	return unix.EpollCtl(p.Epfd, unix.EPOLL_CTL_DEL, fd, nil)
}

//Close 关闭FD
func (p *Poller) Close() error {
	return unix.Close(p.Epfd)
}

//GetConnectMgr .
func (p *Poller) GetConnectMgr() iface.IConnectManager {
	return p.ConnectMgr
}

//ProceedWrite 将之前未发送完毕的数据，继续发送出去
func (p *Poller) ProceedWrite(conn iface.IConnect) error {

	// 1. 获取一个待发送的数据
	dataBuff, empty := conn.GetWriteBuff()

	// 2. 队列中没有未发送完毕的数据，将当前连接改为可读事件
	if empty {

		// 更改为可读状态
		if err := p.ModRead(conn.GetFd(), conn.GetID()); err != nil {
			return err
		}

		// 同步状态
		conn.SetState(common.EPollIN)

		return nil
	}

	// 3. 发送
	n, err := unix.Write(conn.GetFd(), dataBuff)

	//fmt.Printf("dataBuff %d empty %v 已发送[%d] 剩余[%d]\n", len(dataBuff), empty, n, len(dataBuff)-n)
	if err != nil {
		return err
	}

	// 设置writeBuff
	conn.SetWriteBuff(dataBuff[n:])
	return nil
}

//ClearByConn 统一入口关闭某个连接
func (p *Poller) ClearByConn(conn iface.IConnect) {
	_ = conn.Close()           // 断开连接
	_ = p.Remove(conn.GetFd()) // 删除事件订阅
	p.ConnectMgr.Remove(conn)
}
