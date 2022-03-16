package server

import "github.com/ikilobyte/netman/iface"

type websocketProtocol struct {
	*BaseConnect
}

func (c *websocketProtocol) Send(msgID uint32, bs []byte) (int, error) {
	panic("implement me")
}

func (c *websocketProtocol) Close() error {

	return nil
}

func newWebsocketProtocol(baseConnect *BaseConnect) iface.IConnect {
	return &websocketProtocol{BaseConnect: baseConnect}
}
