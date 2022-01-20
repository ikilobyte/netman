package server

import (
	"github.com/ikilobyte/netman/iface"
	"github.com/ikilobyte/netman/util"
)

type RouterMgr struct {
	inner map[uint32]iface.IRouter
}

func NewRouterMgr() *RouterMgr {
	return &RouterMgr{
		inner: make(map[uint32]iface.IRouter),
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

func (r *RouterMgr) Do(request iface.IRequest) error {
	// 根据msgID获取router
	router, err := r.Get(request.GetMessage().ID())
	if err != nil {
		return err
	}

	// TODO 是否需要使用 worker poll
	go router.Do(request)
	return nil
}
