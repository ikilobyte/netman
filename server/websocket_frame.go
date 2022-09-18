package server

import (
	"bytes"
	"syscall"
	"unicode/utf8"

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

	// 判断当前分帧是否完毕，没有读取到完毕的长度的话，需要继续读取
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

		opcode := c.opcode

		// 重置状态
		c.reset()

		// 所有分帧完毕
		if c.final == 1 {
			//fmt.Println("c.continueBuffer.Bytes()",
			//	c.continueBuffer.Bytes(),
			//	c.packetBuffer.Bytes(),
			//	c.packetBuffer.String(),
			//	opcode, // 只有当前是延续帧的时候，才需要处理
			//	c.continueBuffer.Len(),
			//)

			// 是一个延续帧，且保存了之前的数据
			if opcode == CONTINUATION && c.continueBuffer.Len() >= 1 {
				c.continueBuffer.Write(c.packetBuffer.Bytes())
				c.packetBuffer = c.continueBuffer

				// 重置延续帧的buffer
				c.continueBuffer = bytes.NewBuffer([]byte{})
			}

			// 文本模式必须是UTF-8编码的，需要判断一个完整的包，而不是分帧
			if c.messageMode == TEXTMODE && !utf8.Valid(c.packetBuffer.Bytes()) {
				return nil, util.WebsocketMustUtf8
			}

			message := &util.Message{
				MsgID:       c.msgID,
				DataLen:     uint32(c.packetBuffer.Len()),
				Data:        c.packetBuffer.Bytes(),
				Opcode:      c.messageMode,
				IsWebSocket: true,
			}

			// 重置这个消息类型，只有这几种类型的时候才可以重置
			if opcode == CONTINUATION || opcode == TEXTMODE || opcode == BINMODE {
				c.messageMode = 0
			}

			// 继续重置状态
			c.msgID += 1
			c.packetBuffer = bytes.NewBuffer([]byte{})
			return message, nil
		} else {

			// 延续帧才需要处理的
			c.continueBuffer.Write(c.packetBuffer.Bytes())

			// 这是一个完整的
			c.packetBuffer.Reset()
		}
	}

	return nil, syscall.EAGAIN
}
