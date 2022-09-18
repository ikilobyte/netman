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

//ModWrite 将事件修改为写
func (p *Poller) ModWrite(fd, connID int) error {

	// 删除读事件
	_, err := unix.Kevent(p.Epfd, []unix.Kevent_t{
		{
			Ident:  uint64(fd),
			Filter: unix.EVFILT_READ,
			Flags:  unix.EV_DELETE,
			Fflags: 0,
			Data:   0,
			Udata:  nil,
		},
	}, nil, nil)

	if err != nil {
		return err
	}

	// 添加写事件
	return p.AddWrite(fd, connID)
}

//ModRead 将事件修改为读
func (p *Poller) ModRead(fd, connID int) error {
	// 删除写事件
	_, err := unix.Kevent(p.Epfd, []unix.Kevent_t{
		{
			Ident:  uint64(fd),
			Filter: unix.EVFILT_WRITE,
			Flags:  unix.EV_DELETE,
			Fflags: 0,
			Data:   0,
			Udata:  nil,
		},
	}, nil, nil)

	if err != nil {
		return err
	}

	// 添加读事件
	return p.AddRead(fd, connID)
}

//Wait 这里处理的是socket的读
func (p *Poller) Wait(emitCh chan<- iface.IContext) {

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
				event     = p.Events[i]
				connFd    = int(event.Ident)
				conn      iface.IConnect
				connEvent iface.IConnectEvent
			)

			// 1、通过connID获取conn实例
			if conn = p.ConnectMgr.Get(connFd); conn == nil {
				// 断开连接
				_ = unix.Close(connFd)
				_ = p.Remove(connFd)
				continue
			}

			connEvent = conn.(iface.IConnectEvent)

			// 判断是否为写事件
			if event.Filter == unix.EVFILT_WRITE {
				if err := connEvent.ProceedWrite(); err != nil {
					// 断开连接
					_ = conn.Close()
					util.Logger.Errorf("kqueue proceed write error %v", err)
					continue
				}
				continue
			}

			// 1、判断是否开启tls
			if conn.GetTLSEnable() && conn.GetHandshakeCompleted() == false {

				tlsLayer := conn.GetTLSLayer()
				if err := tlsLayer.Handshake(); err != nil {
					// 断开连接
					_ = conn.Close()
					util.Logger.Errorf("tls handshake error %v", err)
					continue
				}
				// 1、设置状态
				conn.SetHandshakeCompleted()

				// 2、设置当前FD为非阻塞模式
				if err := unix.SetNonblock(connFd, true); err != nil {
					_ = conn.Close()
					continue
				}
			}

			// 2、读取一个完整的包
			message, err := connEvent.DecodePacket()
			if err != nil {

				switch err {
				case io.EOF, util.HeadBytesLengthFail, util.BodyLenExceedLimit:
					// 断开连接操作
					_ = conn.Close()
				case
					util.WebsocketOpcodeFail,
					util.WebsocketRsvFail,
					util.WebsocketCtrlMessageMustNotFragmented,
					util.WebsocketPingPayloadOversize:
					_ = conn.(iface.IWebsocketCloser).CloseCode(1002, "protocol error.")
				default:
					continue
				}
			}

			if message == nil {
				continue
			}

			// 3、将消息传递出去，交给worker处理（websocket是可以发送payload长度为0的消息）
			if message.Len() <= 0 && message.IsWebsocket() == false {
				continue
			}
			emitCh <- util.NewContext(util.NewRequest(conn, message, p.ConnectMgr))
		}
	}
}

func (p *Poller) Remove(fd int) error {
	return nil
}

func (p *Poller) Close() error {
	return unix.Close(p.Epfd)
}

//GetConnectMgr .
func (p *Poller) GetConnectMgr() iface.IConnectManager {
	return p.ConnectMgr
}
