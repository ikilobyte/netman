package common

type ConnectState int

const (
	Offline ConnectState = iota
	OnLine
	EPollOUT
	EPollIN
)

type ApplicationMode = int

const (
	RouterMode ApplicationMode = iota
	WebsocketMode
)
