package iface

type IPoller interface {
	AddRead(fd int) error
	AddWrite(fd int) error
	Wait()
	Remove(fd int) error
}
