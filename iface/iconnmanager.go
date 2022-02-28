package iface

type IConnectManager interface {
	Get(connFD int) IConnect
	Add(conn IConnect) int
	GetAll() []IConnect
	Remove(conn IConnect)
	Len() int
	ClearByEpFd(epfd int)
	ClearAll()
	HeartbeatCheck()
}
