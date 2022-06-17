package server

import "github.com/ikilobyte/netman/iface"

type MiddlewareGroup struct {
	middlewares []iface.MiddlewareFunc
	routers     map[uint32]iface.IRouter
}

//newMiddlewareGroup .
func newMiddlewareGroup(callables ...iface.MiddlewareFunc) iface.IMiddlewareGroup {
	group := &MiddlewareGroup{
		middlewares: callables,
	}

	return group
}

//AddRouter 添加路由
func (m *MiddlewareGroup) AddRouter(msgID uint32, router iface.IRouter) {
	m.routers[msgID] = router
}
