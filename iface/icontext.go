package iface

type IContext interface {
	GetRequest() IRequest
	GetConnect() IConnect
	GetMessage() IMessage
	Set(key, value interface{})
	Get(key interface{}) interface{}
}
