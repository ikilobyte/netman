package iface

type IRequest interface {
	GetConnect() IConnect
	GetMessage() IMessage
}
