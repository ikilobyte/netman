package iface

type IConnect interface {
	GetFd() int
	GetID() int
	Close() error
}
