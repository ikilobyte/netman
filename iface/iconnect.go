package iface

import "net"

type IConnect interface {
	Read(bs []byte) (int, error)
	GetFd() int
	GetID() int
	Close() error
	GetPacker() IPacker
	Write(msgID uint32, bs []byte) (int, error)
	GetAddress() net.Addr
	SetEpFd(epfd int)
	GetEpFd() int
	SetPoller(poller IPoller)
	SetWriteBuff([]byte)
	GetWriteBuff() []byte
}
