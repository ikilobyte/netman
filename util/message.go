package util

//Message 收到数据的封装
type Message struct {
	MsgID   uint32 // 消息ID
	DataLen uint32 // 消息长度
	Data    []byte // 消息
	ReadNum int
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

func (m *Message) SetReadNum(n int) {
	m.ReadNum = n
}

func (m *Message) GetReadNum() int {
	return m.ReadNum
}
