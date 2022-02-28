package util

import (
	"time"

	"github.com/ikilobyte/netman/iface"
)

type Request struct {
	message    iface.IMessage
	connect    iface.IConnect
	connectMgr iface.IConnectManager
}

func NewRequest(connect iface.IConnect, message iface.IMessage, connectMgr iface.IConnectManager) *Request {
	connect.SetLastMessageTime(time.Now())
	return &Request{
		connect:    connect,
		message:    message,
		connectMgr: connectMgr,
	}
}

//GetConnect 获取连接
func (r *Request) GetConnect() iface.IConnect {
	return r.connect
}

//GetMessage 获取消息
func (r *Request) GetMessage() iface.IMessage {
	return r.message
}

//GetConnects 获取所有的connect
func (r *Request) GetConnects() []iface.IConnect {
	return r.connectMgr.GetConnects()
}
