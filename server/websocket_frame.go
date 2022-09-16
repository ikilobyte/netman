package server

import (
	"bytes"
	"github.com/ikilobyte/netman/iface"
	"github.com/ikilobyte/netman/util"
)

//nextFrame 读取帧数据
func (c *websocketProtocol) nextFrame() (iface.IMessage, error) {

	// 解析payload
	rLen := c.fragmentLength - uint(c.rBuffer.Len())
	payloadBuffer := make([]byte, rLen)
	n, err := c.readData(payloadBuffer)

	// 包体长度是0
	if c.fragmentLength != 0 {
		if n <= 0 || err != nil {
			return nil, err
		}
	}

	// 保存到buffer中，非阻塞时下次可以继续追加
	c.rBuffer.Write(payloadBuffer[:n])

	// 判断当前分帧是否完毕
	if uint(c.rBuffer.Len()) == c.fragmentLength {

		var decodeBuffer []byte
		fragmentBuffer := c.rBuffer.Bytes()
		if len(c.masks) == 4 {
			decodeBuffer = make([]byte, c.fragmentLength)
			for i := 0; i < c.rBuffer.Len(); i++ {
				decodeBuffer[i] = fragmentBuffer[i] ^ c.masks[i%4]
			}
		} else {
			decodeBuffer = fragmentBuffer
		}

		c.packetBuffer.Write(decodeBuffer)

		// 重置状态
		c.reset()

		// 所有分帧完毕
		if c.final == 1 {
			message := &util.Message{
				MsgID:       c.msgID,
				DataLen:     uint32(c.packetBuffer.Len()),
				Data:        c.packetBuffer.Bytes(),
				Opcode:      c.messageMode,
				IsWebSocket: true,
			}

			// 重置这个消息类型
			c.messageMode = 0

			// 继续重置状态
			c.msgID += 1
			c.packetBuffer = bytes.NewBuffer([]byte{})
			return message, nil
		}
	}

	return nil, nil
}
