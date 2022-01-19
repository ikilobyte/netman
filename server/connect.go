package server

import (
	"golang.org/x/sys/unix"
)

//Connect 每个连接的具体定义
type Connect struct {
	id int           // 自定义生成的ID
	fd int           // 系统分配的fd
	Sa unix.Sockaddr //
}

//NewConnect 构造一个连接
func NewConnect(id int, fd int, sa unix.Sockaddr) *Connect {
	return &Connect{id: id, fd: fd, Sa: sa}
}

//GetID 获取连接ID
func (c *Connect) GetID() int {
	return c.id
}

//GetFd 获取系统分配的fd
func (c *Connect) GetFd() int {
	return c.fd
}

//Close 断开连接
func (c *Connect) Close() error {
	return unix.Close(c.fd)
}
