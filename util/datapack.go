package util

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"golang.org/x/sys/unix"

	"github.com/ikilobyte/netman/iface"
)

//DataPacker 可以自行实现IPacker，可以按照自己的协议格式来处理
type DataPacker struct{}

func NewDataPacker() *DataPacker {
	return &DataPacker{}
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

	// 读取msgID
	if err := binary.Read(dataBuff, binary.LittleEndian, &msgId); err != nil {
		return nil, err
	}

	return &Message{
		MsgID:   msgId,
		DataLen: dataLen,
	}, nil
}

//ReadFull 调用这个方法可以获取一个完整的message
func (d *DataPacker) ReadFull(fd int) (iface.IMessage, error) {

	// 读取头部8个字节
	var headBytes = make([]byte, 8)
	n, err := unix.Read(fd, headBytes)

	// 连接断开
	if n == 0 {
		return nil, ConnectClosed
	}

	// 读取数据有误
	if err != nil {
		return nil, err
	}

	// 包头数据读取有误
	if n != len(headBytes) {
		return nil, HeadBytesLengthFail
	}

	// 解包
	message, err := d.UnPack(headBytes)
	if err != nil {
		return nil, err
	}

	// 继续读取剩余的数据
	dataBuff := make([]byte, message.Len())

	for {
		n, err = unix.Read(fd, dataBuff)

		// 连接断开
		if n == 0 {
			return nil, ConnectClosed
		}

		// 读取数据有误
		if err != nil {
			// 数据读完了
			if err == unix.EAGAIN {
				break
			}

			// 还没有读完
			if err == unix.EINTR {
				continue
			}

			return nil, err
		}

	}

	fmt.Println(dataBuff)
	// 设置数据
	//message.SetData(dataBuff)

	return message, nil
}
