package util

import "github.com/ikilobyte/netman/iface"

type Request struct {
	message iface.IMessage
	connect iface.IConnect
}

func NewRequest(connect iface.IConnect, message iface.IMessage) *Request {
	return &Request{connect: connect, message: message}
}

//GetConnect 获取连接
func (r *Request) GetConnect() iface.IConnect {
	return r.connect
}

//GetMessage 获取消息
func (r *Request) GetMessage() iface.IMessage {
	return r.message
}
