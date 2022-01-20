package iface

//IServer Server抽象层
type IServer interface {
	Start()
	Stop()
	AddRouter(msgID uint32, router IRouter)
}
