package util

import (
	"bytes"
	"encoding/binary"

	"github.com/ikilobyte/netman/iface"
)

//DataPacker 可以自行实现IPacker，可以按照自己的协议格式来处理
type DataPacker struct {
	maxBodyLength uint32
}

func NewDataPacker() *DataPacker {
	return &DataPacker{}
}

//SetMaxBodyLength .
func (d *DataPacker) SetMaxBodyLength(maxBodyLength uint32) {
	d.maxBodyLength = maxBodyLength
}

//Pack 封包格式：data长度(4字节)msgID(4字节)data
func (d *DataPacker) Pack(msgID uint32, data []byte) ([]byte, error) {

	buff := bytes.NewBuffer([]byte{})

	// 写入data长度
	if err := binary.Write(buff, binary.LittleEndian, uint32(len(data))); err != nil {
		return nil, err
	}

	// 写入msgID
	if err := binary.Write(buff, binary.LittleEndian, msgID); err != nil {
		return nil, err
	}

	// 写入data
	if err := binary.Write(buff, binary.LittleEndian, data); err != nil {
		return nil, err
	}

	return buff.Bytes(), nil
}

//UnPack 解包数据（传到这里的只有前8个字节），后续的data部分需要Read读取
func (d *DataPacker) UnPack(bs []byte) (iface.IMessage, error) {

	dataBuff := bytes.NewBuffer(bs)
	var (
		dataLen uint32
		msgId   uint32
	)

	// 读取数据长度
	if err := binary.Read(dataBuff, binary.LittleEndian, &dataLen); err != nil {
		return nil, err
	}

	// 判断长度是否超过限制
	if d.maxBodyLength > 0 && dataLen > d.maxBodyLength {
		Logger.Errorln(BodyLenExceedLimit)
		return nil, BodyLenExceedLimit
	}

	// 读取msgID
	if err := binary.Read(dataBuff, binary.LittleEndian, &msgId); err != nil {
		return nil, err
	}

	return &Message{
		MsgID:   msgId,
		DataLen: dataLen,
	}, nil
}

//GetHeaderLength 获取头部长度
func (d *DataPacker) GetHeaderLength() uint32 {
	return 8
}

//readData 读取数据
func (d *DataPacker) readData(connect iface.IConnect, bs []byte) (int, error) {
	if connect.GetTLSEnable() {
		return connect.GetTLSLayer().Read(bs)
	}
	return connect.Read(bs)
}
