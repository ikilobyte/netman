package server

import (
	"github.com/ikilobyte/netman/iface"
	"github.com/ikilobyte/netman/util"
)

type RouterMgr struct {
	inner             map[uint32]iface.IRouter          // 所有的路由
	routeMiddleware   map[uint32][]iface.MiddlewareFunc // 路由中间件
	globalMiddlewares []iface.MiddlewareFunc            // 全局中间件
}

//NewRouterMgr 中间件执行顺序 globalMiddleware -> routerMiddleware
func NewRouterMgr() *RouterMgr {
	return &RouterMgr{
		inner:             make(map[uint32]iface.IRouter),
		routeMiddleware:   make(map[uint32][]iface.MiddlewareFunc),
		globalMiddlewares: make([]iface.MiddlewareFunc, 0),
	}
}

//Add 添加路由
func (r *RouterMgr) Add(msgID uint32, router iface.IRouter) {
	r.inner[msgID] = router
}

//Get 根据msgID获取路由
func (r *RouterMgr) Get(msgID uint32) (iface.IRouter, error) {

	router, ok := r.inner[msgID]
	if ok {
		return router, nil
	}
	return nil, util.RouterNotFound
}

func (r *RouterMgr) Do(ctx iface.IContext) error {
	// 根据msgID获取router
	request := ctx.GetRequest()
	router, err := r.Get(request.GetMessage().ID())
	if err != nil {
		return err
	}

	middlewares := make([]iface.MiddlewareFunc, 0)

	// 全局中间件
	middlewares = append(middlewares, r.globalMiddlewares...)

	// 路由中间件
	middlewares = append(middlewares, r.routeMiddleware[request.GetMessage().ID()]...)

	// 执行
	go func() {
		util.NewPipeline().
			Send(ctx).
			Through(r.Conversion(middlewares)).
			Then(func(value interface{}) interface{} {
				router.Do(value.(iface.IContext).GetRequest())
				return value
			})
	}()

	return nil
}

//Conversion 将中间件转换为stage类型
func (r *RouterMgr) Conversion(middlewares []iface.MiddlewareFunc) []iface.IStage {
	stages := make([]iface.IStage, 0)
	for _, middleware := range middlewares {
		stages = append(stages, &stage{middleware})
	}
	return stages
}

type stage struct {
	middleware iface.MiddlewareFunc
}

func (s *stage) Process(value interface{}, next iface.NextFunc) interface{} {
	return s.middleware(value.(iface.IContext), func(ctx iface.IContext) interface{} {
		return next(ctx)
	})
}
