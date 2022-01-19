package iface

type IMessage interface {
	GetMsgID() uint32
	Bytes() []byte
	String() string
	Len() int
}
