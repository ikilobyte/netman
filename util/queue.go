package util

import (
	"sync"
)

type Queue struct {
	inner  []interface{}
	locker sync.Mutex
}

func NewQueue() *Queue {
	return &Queue{
		inner:  make([]interface{}, 0),
		locker: sync.Mutex{},
	}
}

//Push 加
func (q *Queue) Push(item interface{}) int {
	q.locker.Lock()
	defer q.locker.Unlock()
	q.inner = append(q.inner, item)

	return len(q.inner)
}

//Pop 弹
func (q *Queue) Pop() interface{} {
	q.locker.Lock()
	defer q.locker.Unlock()
	if len(q.inner) <= 0 {
		return nil
	}

	item := q.inner[0]
	q.inner = q.inner[1:]
	return item
}

//Len 获取长度，不准确
func (q *Queue) Len() int {
	return len(q.inner)
}
