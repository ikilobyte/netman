package iface

//IServer Server抽象层
type IServer interface {
	Start()
	Stop()
	GetConnMgr() IConnManager
	Emit(dataBuff []byte)
}
