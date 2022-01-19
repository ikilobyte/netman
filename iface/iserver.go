package iface

//IServer Server抽象层
type IServer interface {
	Start()
	Stop()
	GetConnectMgr() IConnectManager
	GetPacker() IPacker
}