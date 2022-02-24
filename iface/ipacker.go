package iface

//IPacker 数据封装抽象层
type IPacker interface {
	ReadFull(fd int) (IMessage, error)              // 读包
	Pack(msgID uint32, data []byte) ([]byte, error) // 封包
	UnPack([]byte) (IMessage, error)                // 解包
	SetMaxBodyLength(uint32)                        // 设置包体最大长度限制
}
