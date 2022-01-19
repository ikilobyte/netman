package server

import (
	"sync"

	"github.com/ikilobyte/netman/iface"
)

//ConnectManager 所有连接都保存在这里
type ConnectManager struct {
	connects map[int]iface.IConnect
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
