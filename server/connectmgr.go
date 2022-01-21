package server

import (
	"sync"

	"github.com/ikilobyte/netman/iface"
)

//ConnectManager 所有连接都保存在这里
type ConnectManager struct {
	connects map[int]iface.IConnect // connID => Connect
	sync.RWMutex
}

//NewConnectManager 构造一个实例
func NewConnectManager() *ConnectManager {
	return &ConnectManager{
		connects: map[int]iface.IConnect{},
	}
}

//Add 添加一个连接
func (c *ConnectManager) Add(conn iface.IConnect) int {
	c.Lock()
	defer c.Unlock()
	c.connects[conn.GetID()] = conn
	return len(c.connects)
}

//Get 通过connID获取连接实例
func (c *ConnectManager) Get(connID int) iface.IConnect {
	c.Lock()
	defer c.Unlock()
	if conn, ok := c.connects[connID]; ok {
		return conn
	}

	return nil
}

//Remove 删除一个连接
func (c *ConnectManager) Remove(conn iface.IConnect) {
	c.Lock()
	defer c.Unlock()
	delete(c.connects, conn.GetID())
}

//Len 获取有多少个连接
func (c *ConnectManager) Len() int {
	return len(c.connects)
}

//ClearEpFd 删除在这个epfd上管理的所有连接，只有这个epoll出现错误的时候才会调用这个方法
//一份数据最好不要存多个地方，在一个地方统一管理
func (c *ConnectManager) ClearEpFd(epfd int) {

	// TODO 待优化
	c.Lock()
	defer c.Unlock()

	for connID, connect := range c.connects {
		if connect.GetEpFd() != epfd {
			continue
		}

		// 断开连接
		_ = connect.Close()

		// 删除
		delete(c.connects, connID)
	}
}
