package netman

import (
	"fmt"
	"time"
)

type worker struct {
	id        int
	closeCh   chan struct{}
	messageCh chan message
}

//newWorker 创建worker
func newWorker(id int, messageCh chan message) *worker {
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
			fmt.Printf("workerId[%d] recv -> %v\n", w.id, bs)
			time.Sleep(time.Hour)
		}
	}
}
