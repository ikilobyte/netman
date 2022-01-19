package iface

type IPoller interface {
	AddRead(fd, connID int) error
	AddWrite(fd, connID int) error
	Wait()
	Remove(fd int) error
	Close() error
}
