package netman

type message struct {
	dataBuff []byte
}

func newMessage(dataBuff []byte) *message {
	return &message{dataBuff: dataBuff}
}

func (m *message) GetBytes() []byte {
	return m.dataBuff
}
