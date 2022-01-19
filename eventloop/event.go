package eventloop

import (
	"fmt"
	"log"

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
func (e *EventLoop) Init(connectMgr iface.IConnectManager) {

	for i := 0; i < e.Num; i++ {
		poller, err := NewPoller(connectMgr)
		if err != nil {
			log.Panicln("NewPoller err", err)
		}
		e.pollers[i] = poller
	}
}

//Start 执行epoll_wait
func (e *EventLoop) Start() {
	for _, poller := range e.pollers {
		go poller.Wait()
	}
}

//Stop 关闭epoll
func (e *EventLoop) Stop() {
	for _, poller := range e.pollers {
		if err := poller.Close(); err != nil {
			// TODO 待优化
			fmt.Println("poller.Close err", err, poller)
		}
	}
}

//AddRead 添加读事件
func (e *EventLoop) AddRead(conn iface.IConnect) error {
	idx := conn.GetID() % e.Num
	poller := e.pollers[idx]
	return poller.AddRead(conn.GetFd(), conn.GetID())
}

//Remove 删除某个连接
func (e *EventLoop) Remove(conn iface.IConnect) error {
	idx := conn.GetID() % e.Num
	poller := e.pollers[idx]
	return poller.Remove(conn.GetFd())
}
