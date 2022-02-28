package server

import (
	"sync"
	"time"

	"github.com/ikilobyte/netman/iface"
)

//ConnectManager 所有连接都保存在这里
type ConnectManager struct {
	connects map[int]iface.IConnect // connFD => Connect
	options  *Options
	sync.RWMutex
}

//newConnectManager 构造一个实例
func newConnectManager(options *Options) *ConnectManager {

	mgr := &ConnectManager{
		connects: map[int]iface.IConnect{},
		options:  options,
	}

	// 心跳检测
	go mgr.HeartbeatCheck()

	return mgr
}

//Add 添加一个连接
func (c *ConnectManager) Add(conn iface.IConnect) int {
	c.Lock()
	defer c.Unlock()
	c.connects[conn.GetFd()] = conn
	return len(c.connects)
}

//Get 通过connID获取连接实例
func (c *ConnectManager) Get(connFD int) iface.IConnect {
	c.Lock()
	defer c.Unlock()
	if conn, ok := c.connects[connFD]; ok {
		return conn
	}

	return nil
}

//Remove 删除一个连接
func (c *ConnectManager) Remove(conn iface.IConnect) {
	c.Lock()
	defer c.Unlock()
	delete(c.connects, conn.GetFd())
}

//Len 获取有多少个连接
func (c *ConnectManager) Len() int {
	return len(c.connects)
}

//ClearByEpFd 删除在这个epfd上管理的所有连接，只有这个epoll出现错误的时候才会调用这个方法
//一份数据最好不要存多个地方，在一个地方统一管理
func (c *ConnectManager) ClearByEpFd(epfd int) {

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

//ClearAll 清除所有连接
func (c *ConnectManager) ClearAll() {
	c.Lock()
	defer c.Unlock()

	for _, connect := range c.connects {
		_ = connect.Close()
	}
	c.connects = make(map[int]iface.IConnect)
}

//HeartbeatCheck 心跳检测
func (c *ConnectManager) HeartbeatCheck() {

	if int(c.options.HeartbeatCheckInterval) <= 0 || int(c.options.HeartbeatIdleTime) < 0 {
		return
	}

	idleTime := int64(c.options.HeartbeatIdleTime / time.Second)

	// 在遍历所有连接时，不会加锁，不能影响到正常操作，但可能会有
	for {
		select {
		case now := <-time.Tick(c.options.HeartbeatCheckInterval):
			for _, connect := range c.connects {
				lastMessageTime := connect.GetLastMessageTime()
				if now.Unix()-lastMessageTime.Unix() < idleTime {
					continue
				}

				// 强制断开连接，会正常执行OnClose回调
				_ = connect.Close()
				c.Remove(connect)
				_ = connect.GetPoller().Remove(connect.GetFd())
			}
		}
	}
}

//GetConnects 获取所有连接
func (c *ConnectManager) GetConnects() []iface.IConnect {
	connects := make([]iface.IConnect, 0, c.Len())
	for _, connect := range c.connects {
		connects = append(connects, connect)
	}
	return connects
}
