package util

import (
	"sync"

	"github.com/ikilobyte/netman/iface"
)

type connManager struct {
	connections map[int]iface.IConnection
	sync.RWMutex
}

func NewConnManager() *connManager {
	return &connManager{
		connections: map[int]iface.IConnection{},
	}
}

func (c *connManager) Add(conn iface.IConnection) int {
	c.Lock()
	defer c.Unlock()
	c.connections[conn.GetID()] = conn
	return len(c.connections)
}

func (c *connManager) Remove(id int) {
	c.Lock()
	defer c.Unlock()
	delete(c.connections, id)
}

func (c *connManager) Len() int {
	return len(c.connections)
}
