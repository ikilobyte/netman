package iface

type IConnectManager interface {
	Get(connFD int) IConnect
	Add(conn IConnect) int
	GetConnects() []IConnect
	Remove(conn IConnect)
	Len() int
	ClearByEpFd(epfd int)
	ClearAll()
	HeartbeatCheck()
}
