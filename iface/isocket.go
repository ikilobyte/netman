package iface

type ISocket interface {
	GetFd() int
	MakeFd()
	Bind() (err error)
	Listen() (err error)
	Accept() (IConnection, error)
}
