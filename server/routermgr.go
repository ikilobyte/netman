package server

import (
	"fmt"

	"github.com/ikilobyte/netman/common"
	"github.com/ikilobyte/netman/iface"
	"github.com/ikilobyte/netman/util"
)

type RouterMgr struct {
	inner             map[uint32]iface.IRouter          // 所有的路由
	routeMiddleware   map[uint32][]iface.MiddlewareFunc // 路由中间件
	globalMiddlewares []iface.MiddlewareFunc            // 全局中间件
	middlewareGroup   []iface.IMiddlewareGroup
}

//NewRouterMgr 中间件执行顺序 globalMiddleware -> routerMiddleware
func NewRouterMgr() *RouterMgr {
	return &RouterMgr{
		inner:             make(map[uint32]iface.IRouter),
		routeMiddleware:   make(map[uint32][]iface.MiddlewareFunc),
		globalMiddlewares: make([]iface.MiddlewareFunc, 0),
		middlewareGroup:   make([]iface.IMiddlewareGroup, 0),
	}
}

//Add 添加路由
func (r *RouterMgr) Add(msgID uint32, router iface.IRouter) {
	r.inner[msgID] = router
}

//NewGroup 中间一个中间件组
func (r *RouterMgr) NewGroup(callable iface.MiddlewareFunc, more ...iface.MiddlewareFunc) iface.IMiddlewareGroup {
	group := newMiddlewareGroup(append(more, callable)...)
	r.middlewareGroup = append(r.middlewareGroup, group)
	return group
}

//ResolveGroup 处理路由分组的数据
func (r *RouterMgr) ResolveGroup() error {
	for _, group := range r.middlewareGroup {
		for routerID, router := range group.GetRouters() {
			r.routeMiddleware[routerID] = group.GetMiddlewares()
			r.Add(routerID, router)
		}
	}
	return nil
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

	// 执行方法
	router.Do(request)
	return nil
}

//Dispatch 路由分发和中间件执行
func (r *RouterMgr) Dispatch(ctx iface.IContext, options *Options) {

	request := ctx.GetRequest()

	// 合并中间件
	middlewares := make([]iface.MiddlewareFunc, 0)

	// 全局中间件
	middlewares = append(middlewares, r.globalMiddlewares...)

	// 路由中间件
	middlewares = append(middlewares, r.routeMiddleware[request.GetMessage().ID()]...)

	// 先执行中间件
	util.NewPipeline().
		Send(ctx).
		Through(r.Conversion(middlewares)).
		Then(func(value interface{}) interface{} {

			var err error

			// 当前Server是websocket协议
			if options.Application == common.WebsocketMode {
				options.WebsocketHandler.Message(ctx.GetRequest())
				return err
			}

			// TCP协议
			if err = r.Do(ctx); err != nil {
				util.Logger.Infoln(fmt.Errorf("do handler err %s", err))
			}

			return err
		})
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
