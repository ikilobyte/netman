package iface

type IMessage interface {
	ID() uint32
	Bytes() []byte
	String() string
	Len() int
	SetData([]byte)
	GetOpcode() uint8
	IsWebsocket() bool
}
