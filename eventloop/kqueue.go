// +build darwin

package eventloop

// TODO
//poller macos kqueue
type poller struct {
}

func newPoller() *poller {
	return &poller{}
}

func (p *poller) AddRead(fd, pad int) error {
	panic("implement me")
}

func (p *poller) AddWrite(fd, pad int) error {
	panic("implement me")
}

func (p *poller) Wait() {
	panic("implement me")
}

func (p *poller) Remove(fd int) error {
	panic("implement me")
}
