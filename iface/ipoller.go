package iface

type IPoller interface {
	AddRead(fd, pad int) error
	AddWrite(fd, pad int) error
	Wait()
	Remove(fd int) error
	Close() error
}
