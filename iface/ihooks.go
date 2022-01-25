package iface

type IHooks interface {
	OnOpen(connect IConnect)
	OnClose(connect IConnect)
}
