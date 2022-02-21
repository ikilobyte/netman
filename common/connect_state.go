package common

type ConnectState int

const (
	Offline ConnectState = iota
	OnLine
	EPollOUT
	EPollIN
)
