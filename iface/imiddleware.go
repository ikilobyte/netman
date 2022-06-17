package iface

type Next = func(ctx IContext) interface{}
type MiddlewareFunc = func(ctx IContext, next Next) interface{}

type IMiddlewareGroup interface {
	AddRouter(msgID uint32, router IRouter)
	GetMiddlewares() []MiddlewareFunc
	GetRouters() map[uint32]IRouter
}
