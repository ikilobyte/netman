package util

//Message 收到数据的封装
type Message struct {
	MsgID   uint32 // 消息ID
	DataLen uint32 // 消息长度
	Data    []byte // 消息
}

func NewMessage(data []byte) *Message {
	return &Message{Data: data, DataLen: uint32(len(data))}
}

func (m *Message) GetMsgID() uint32 {
	return m.MsgID
}

func (m *Message) String() string {
	return string(m.Data)
}

func (m *Message) Bytes() []byte {
	return m.Data
}

func (m *Message) Len() int {
	return int(m.DataLen)
}
