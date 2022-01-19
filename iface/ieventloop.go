package iface

//IEventLoop 事件循环抽象层，所有的epoll都是通过这个来操作
type IEventLoop interface {
	Init()  // 初始化，也就是创建epoll
	Start() // 开启事件循环，也就是所有的epoll执行epoll_wait
	Stop()  // 停止
	AddRead(conn IConnect) error
	Remove(conn IConnect) error
}
