package iface

type IPoller interface {
	AddRead(fd int, connID int) error
	AddWrite(fd, connID int) error
	ModWrite(fd, connID int) error
	ModRead(fd, connId int) error
	Wait(emitCh chan<- IContext)
	Remove(fd int) error
	Close() error
	GetConnectMgr() IConnectManager
}
