package iface

type IConnection interface {
	GetFd() int
	GetID() int
}
