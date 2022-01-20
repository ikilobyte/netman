package iface

//IRouter 路由抽象，根据业务场景实现这个接口即可，通过msgID和router对应
type IRouter interface {
	Do(request IRequest)
}
