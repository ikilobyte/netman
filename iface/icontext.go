package iface

type IContext interface {
	GetRequest() IRequest
	GetConnect() IConnect
	Set(key, value interface{})
	Get(key interface{}) interface{}
}
