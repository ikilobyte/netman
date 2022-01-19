package server

import (
	"fmt"

	"github.com/ikilobyte/netman/iface"
)

type worker struct {
	id        int
	closeCh   chan struct{}
	messageCh chan iface.IMessage
}

//newWorker 创建worker
func newWorker(id int, messageCh chan iface.IMessage) *worker {
	return &worker{
		id:        id,
		closeCh:   make(chan struct{}),
		messageCh: messageCh,
	}
}

//Start 启动worker，等待处理任务
func (w *worker) Start() {
	for {
		select {
		case <-w.closeCh:
			return
		case bs := <-w.messageCh:
			fmt.Printf("workerId[%d] recv -> %v\n", w.id, bs.String())
		}
	}
}
