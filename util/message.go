package util

//Message 收到数据的封装
type Message struct {
	MsgID       uint32 // 消息ID
	DataLen     uint32 // 消息长度
	Data        []byte // 消息
	IsWebSocket bool   // 是否为websocket协议
	Opcode      uint8  // 操作码
}

func (m *Message) ID() uint32 {
	return m.MsgID
}

func (m *Message) String() string {
	return string(m.Data)
}

//Bytes 获取Bytes
func (m *Message) Bytes() []byte {
	return m.Data
}

//Len 获取长度
func (m *Message) Len() int {
	return int(m.DataLen)
}

//SetData 设置数据
func (m *Message) SetData(bytes []byte) {
	m.Data = bytes
	m.DataLen = uint32(len(bytes)) // 更新长度
}

//GetOpcode 获取操作码，通过这个可以判断是否为websocket消息
func (m *Message) GetOpcode() uint8 {
	return m.Opcode
}

func (m *Message) IsWebsocket() bool {
	return m.IsWebSocket
}
