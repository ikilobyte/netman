package iface

type IConnManager interface {
	Add(conn IConnection) int // 新增
	Remove(id int)            // 删除
	Len() int                 // 长度
}
