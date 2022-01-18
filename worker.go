package netman

import "fmt"

type worker struct {
	id        int
	closeCh   chan struct{}
	messageCh chan []byte
}

//newWorker 创建worker
func newWorker(id int) *worker {
	return &worker{
		id:        id,
		closeCh:   make(chan struct{}),
		messageCh: make(chan []byte, 100),
	}
}

//Start 启动worker，等待处理任务
func (w *worker) Start() {
	for {
		select {
		case <-w.closeCh:
			return
		case bs := <-w.messageCh:
			fmt.Println("bytes", bs)
		}
	}
}
