package server

import (
	"fmt"

	"github.com/ikilobyte/netman/iface"
)

type BaseRouter struct{}

func (b *BaseRouter) Do(request iface.IRequest) {
	conn := request.GetConnect()
	message := request.GetMessage()
	fmt.Println(conn, message)
}
