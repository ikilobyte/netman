package util

import (
	"sync"

	"github.com/ikilobyte/netman/iface"
)

type Context struct {
	storage *sync.Map
	request iface.IRequest
}

//NewContext .
func NewContext(request iface.IRequest) iface.IContext {
	return &Context{
		storage: new(sync.Map),
		request: request,
	}
}

func (c *Context) GetRequest() iface.IRequest {
	return c.request
}

func (c *Context) GetConnect() iface.IConnect {
	return c.request.GetConnect()
}

//GetMessage 获取消息
func (c *Context) GetMessage() iface.IMessage {
	return c.request.GetMessage()
}

func (c *Context) Set(key, value interface{}) {
	c.storage.Store(key, value)
}

func (c *Context) Get(key interface{}) interface{} {
	value, ok := c.storage.Load(key)
	if !ok {
		return nil
	}
	return value
}
