package iface

type IAcceptor interface {
	Run(fd int, loop IEventLoop) error
	Exit()
	IncrementID() int
	Close()
}
