package eventloop

import (
	"github.com/ikilobyte/netman/iface"
)

type EventLoop struct {
	Num     int       // 数量
	pollers []*Poller // 所以的poller
}

func NewEventLoop(num int) *EventLoop {
	return &EventLoop{
		Num:     num,
		pollers: make([]*Poller, num),
	}
}

//Init 初始化poller
func (e *EventLoop) Init(connectMgr iface.IConnectManager) error {

	for i := 0; i < e.Num; i++ {
		poller, err := NewPoller(connectMgr)
		if err != nil {
			return err
		}
		e.pollers[i] = poller
	}
	return nil
}

//Start 执行epoll_wait
func (e *EventLoop) Start(emitCh chan<- iface.IRequest) {
	for _, poller := range e.pollers {
		go poller.Wait(emitCh)
	}
}

//Stop 关闭epoll
func (e *EventLoop) Stop() {
	for _, poller := range e.pollers {
		_ = poller.Close()
	}
}

//AddRead 添加读事件
func (e *EventLoop) AddRead(conn iface.IConnect) error {
	idx := conn.GetID() % e.Num
	poller := e.pollers[idx]
	if err := poller.AddRead(conn.GetFd(), conn.GetID()); err != nil {
		return err
	}

	// TODO 不应该暴露出去
	conn.SetEpFd(poller.Epfd)
	return nil
}

//Remove 删除某个连接
func (e *EventLoop) Remove(conn iface.IConnect) error {
	idx := conn.GetID() % e.Num
	poller := e.pollers[idx]
	return poller.Remove(conn.GetFd())
}
