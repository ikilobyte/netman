package iface

type IConnect interface {
	Read(bs []byte) (int, error)
	GetFd() int
	GetID() int
	Close() error
	GetPacker() IPacker
	Write(msgID uint32, bs []byte) (int, error)
}
