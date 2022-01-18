package util

import (
	"golang.org/x/sys/unix"
)

type connect struct {
	id int           // 自定义生成的ID
	fd int           // 系统分配的fd
	Sa unix.Sockaddr //
}

//GetID 获取连接ID
func (c *connect) GetID() int {
	return c.id
}

//GetFd 获取系统分配的fd
func (c *connect) GetFd() int {
	return c.fd
}

//NewConnect 创建一个新的连接
func NewConnect(id int, fd int, sa unix.Sockaddr) *connect {
	return &connect{id: id, fd: fd, Sa: sa}
}
