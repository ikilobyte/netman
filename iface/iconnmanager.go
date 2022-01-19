package iface

type IConnectManager interface {
	Add(conn IConnect) int
	Remove(conn IConnect)
	Len() int
}
