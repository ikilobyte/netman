package util

import (
	"bytes"
	"encoding/binary"
	"io"
	"time"

	"golang.org/x/sys/unix"

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

//ReadFull 调用这个方法可以获取一个完整的message
func (d *DataPacker) ReadFull(connect iface.IConnect) (iface.IMessage, error) {

	// 读取头部8个字节
	headBytes := make([]byte, 8)
	n, err := d.readData(connect, headBytes)

	// 连接断开
	if n == 0 && err == io.EOF {
		return nil, io.EOF
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
	dataBuff := bytes.NewBuffer([]byte{})
	readLen := message.Len()
	readTotal := 0 // 记录读了多少次才读完这个包

	// 只有包头
	if readLen <= 0 {
		message.SetData(dataBuff.Bytes())
		message.SetReadNum(0)
		return message, nil
	}

	// TODO telnet测试时会出现问题，一直在读取
	for {

		readBytes := make([]byte, readLen)
		n, err = d.readData(connect, readBytes)
		readTotal += 1

		// 连接断开
		if n == 0 && err == io.EOF {
			return nil, err
		}

		// 读取数据有误
		if err != nil {
			// 还没有读完，继续读数据，这个包可能很大
			if err == unix.EAGAIN || err == unix.EINTR {
				time.Sleep(time.Millisecond * 5)
				continue
			}

			return nil, err
		}

		// 将读取到的数据，保存起来
		dataBuff.Write(readBytes[:n])

		// 判断是否完整
		if dataBuff.Len() == message.Len() {
			break
		} else {
			readLen = message.Len() - dataBuff.Len()
		}
	}

	// 设置数据，返回后可用
	message.SetReadNum(readTotal)
	message.SetData(dataBuff.Bytes())

	return message, nil
}

//readData 读取数据
func (d *DataPacker) readData(connect iface.IConnect, bs []byte) (int, error) {
	if connect.GetTLSEnable() {
		return connect.GetTLSLayer().Read(bs)
	}
	return connect.Read(bs)
}
