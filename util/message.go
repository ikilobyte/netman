package util

type message struct {
	dataBuff []byte
}

func NewMessage(dataBuff []byte) *message {
	return &message{dataBuff: dataBuff}
}

func (m *message) Bytes() []byte {
	return m.dataBuff
}
