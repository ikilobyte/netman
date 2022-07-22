package server

import (
	"bytes"
	"fmt"
	"github.com/ikilobyte/netman/iface"
	"github.com/ikilobyte/netman/util"
	"golang.org/x/sys/unix"
	"io"
)

type routerProtocol struct {
	*BaseConnect
	readBuffer       *bytes.Buffer // 未读取完整的一个数据包
	packDataLength   uint32        // 数据包体长度，如果这个值 == 0，那就是从头开始读取，没有未读取完整的数据
	temporaryMessage iface.IMessage
}

//newRouterProtocol .
func newRouterProtocol(baseConnect *BaseConnect) iface.IConnect {
	connect := &routerProtocol{
		readBuffer:       bytes.NewBuffer([]byte{}),
		packDataLength:   0,
		temporaryMessage: nil,
		BaseConnect:      baseConnect,
	}

	return connect
}

//Close 关闭连接
func (c *routerProtocol) Close() error {

	// 移除事件监听
	_ = c.GetPoller().Remove(c.fd)

	// 从管理类中移除
	c.GetConnectMgr().Remove(c)

	// 关闭连接
	err := unix.Close(c.fd)

	// 重置为0
	c.packDataLength = 0

	// 重置
	c.readBuffer = nil

	// 关闭成功才执行
	if c.hooks != nil && err == nil {
		c.hooks.OnClose(c)
	}

	return err
}

//DecodePacket 解码出一个数据包
func (c *routerProtocol) DecodePacket() (iface.IMessage, error) {

	if c.packDataLength <= 0 {

		// 读取包头
		headBytes := make([]byte, c.packer.GetHeaderLength())
		n, err := c.readData(headBytes)

		// 连接断开
		if n == 0 && err == io.EOF {
			return nil, io.EOF
		}

		// 有错误，可能是 unix.EAGAIN 等错误
		if err != nil {
			// fd有异常
			if err == unix.EBADF || err == unix.EPIPE {
				return nil, io.EOF
			}
			return nil, err
		}

		// 包头数据读取有误
		if n != len(headBytes) {
			return nil, util.HeadBytesLengthFail
		}

		// 解包
		message, err := c.packer.UnPack(headBytes)
		if err != nil {
			return nil, err
		}

		// 包体长度为0
		if message.Len() <= 0 {
			return message, nil
		}

		// 设置长度数据
		c.packDataLength = uint32(message.Len())
		c.temporaryMessage = message
	}

	// 本次读取的最大长度
	// 如果是TLS，那么每次最大读取16384字节即可 https://datatracker.ietf.org/doc/html/rfc8449
	size := c.packDataLength - uint32(c.readBuffer.Len())
	if c.handshakeCompleted && size > 16384 {
		size = 16384
	}
	readBytes := make([]byte, size)

	n, err := c.readData(readBytes)

	// 连接断开
	if n == 0 && err == io.EOF {
		return nil, err
	}

	// 读取数据有误
	if err != nil {

		// FD出现了异常
		if err == unix.EBADF || err == unix.EPIPE {
			return nil, io.EOF
		}

		// 保存到这里
		if n > 0 {
			c.readBuffer.Write(readBytes[:n])
		}
		return nil, err
	}

	// 将读取到的数据保存到这里
	c.readBuffer.Write(readBytes[:n])

	if c.GetHandshakeCompleted() {
		c.tlsRawSize -= n
	}

	// 数据包完整
	if c.readBuffer.Len() == int(c.packDataLength) {

		c.temporaryMessage.SetData(c.readBuffer.Bytes())

		// 重置，每个数据包都是一个互不影响的slice
		c.readBuffer = bytes.NewBuffer([]byte{})

		// 重置包体总长度
		c.packDataLength = 0

		// 重置
		c.tlsRawSize = 0

		return c.temporaryMessage, nil
	} else {

		remain := c.packDataLength - uint32(c.readBuffer.Len())

		// 已完成了TLS握手
		if c.GetHandshakeCompleted() && c.tlsRawSize >= int(remain) {
			fmt.Printf(
				"总长度 %d 已读到 %d 剩余 %d TLSPacketSize %d\n",
				c.packDataLength,
				c.readBuffer.Len(),
				remain,
				c.tlsRawSize,
			)
			return c.DecodePacket()
		}
	}

	return nil, nil
}

//Send 写数据
func (c *routerProtocol) Send(msgID uint32, bytes []byte) (int, error) {

	// 1、封包
	dataPack, err := c.packer.Pack(msgID, bytes)
	if err != nil {
		return 0, err
	}

	// 2、发送
	if c.GetTLSEnable() {
		return c.tlsLayer.Write(dataPack)
	}

	return c.Write(dataPack)
}
