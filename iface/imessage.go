package iface

type IMessage interface {
	ID() uint32
	Bytes() []byte
	String() string
	Len() int
	SetData([]byte)
	SetReadNum(int)
	GetReadNum() int
}
