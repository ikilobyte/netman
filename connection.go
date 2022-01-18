package netman

import (
	"golang.org/x/sys/unix"
)

type connection struct {
	id int           // 自定义生成的ID
	fd int           // 系统分配的fd
	Sa unix.Sockaddr //
}

//GetID 获取连接ID
func (c *connection) GetID() int {
	return c.id
}

//GetFd 获取系统分配的fd
func (c *connection) GetFd() int {
	return c.fd
}

//NewConnection 创建一个新的连接
func newConnection(id int, fd int, sa unix.Sockaddr) *connection {
	return &connection{id: id, fd: fd, Sa: sa}
}
