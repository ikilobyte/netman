package server

import (
	"github.com/ikilobyte/netman/iface"
)

type BaseRouter struct{}

//Do 默认实现
func (b *BaseRouter) Do(request iface.IRequest) {}
