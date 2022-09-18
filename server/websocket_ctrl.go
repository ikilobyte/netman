package server

import (
	"bytes"
	"encoding/binary"
	"net/url"

	"github.com/ikilobyte/netman/iface"
	"github.com/ikilobyte/netman/util"
	"golang.org/x/sys/unix"
)

//push 将封装好的数据推送到客户端
func (c *websocketProtocol) push(dataBuff []byte) (int, error) {

	if c.GetTLSEnable() {
		return c.tlsLayer.Write(dataBuff)
	}
	return c.Write(dataBuff)
}

//encode 封装数据包，不分包，一个包全部推送
func (c *websocketProtocol) encode(firstByte uint8, bs []byte) ([]byte, error) {

	dataBuffer := bytes.NewBuffer([]byte{})

	// 写入第一个字节
	if err := binary.Write(dataBuffer, binary.BigEndian, firstByte); err != nil {
		return nil, err
	}

	totalLen := len(bs)
	if totalLen <= 125 {
		// 写入长度
		if err := binary.Write(dataBuffer, binary.BigEndian, uint8(totalLen)); err != nil {
			return nil, err
		}

	} else if totalLen >= 126 && totalLen <= 65535 {

		// 写入长度
		if err := binary.Write(dataBuffer, binary.BigEndian, uint8(126)); err != nil {
			return nil, err
		}

		// 后续2个字节表示本包的长度
		if err := binary.Write(dataBuffer, binary.BigEndian, uint16(totalLen)); err != nil {
			return nil, err
		}

	} else {

		// 写入长度
		if err := binary.Write(dataBuffer, binary.BigEndian, uint8(127)); err != nil {
			return nil, err
		}

		// 后续8个字节表示本包的长度
		if err := binary.Write(dataBuffer, binary.BigEndian, uint64(totalLen)); err != nil {
			return nil, err
		}
	}

	// 写入数据
	if err := binary.Write(dataBuffer, binary.BigEndian, bs); err != nil {
		return nil, err
	}

	return dataBuffer.Bytes(), nil
}

//remove 从内存中移除
func (c *websocketProtocol) remove() {
	// 移除事件监听
	_ = c.GetPoller().Remove(c.fd)

	// 从管理类中移除
	c.GetConnectMgr().Remove(c)

	// 关闭成功才执行
	if c.hooks != nil {
		c.hooks.OnClose(c) // tcp onclose
	}

	// websocket onclose ，握手成功才执行Close回调
	if c.isHandleShake {
		c.options.WebsocketHandler.Close(c)
	}

	// 重置状态
	c.reset()
	c.packetBuffer = nil
	c.continueBuffer = nil
	c.messageMode = 0

}

//CloseCode 内部关闭，并指定相关code
func (c *websocketProtocol) CloseCode(code uint16, reason string) error {
	data := bytes.NewBuffer([]byte{})
	if err := binary.Write(data, binary.BigEndian, code); err != nil {
		return err
	}

	// 写入reason
	data.WriteString(reason)

	firstByte := uint8(8 | 128)
	encode, _ := c.encode(firstByte, data.Bytes())

	if _, err := c.push(encode); err != nil {
		return err
	}

	c.remove()

	return unix.Close(c.fd)
}

//ping 发送ping包
func (c *websocketProtocol) ping() {
	_, _ = c.Write([]byte{137, 0})
	util.Logger.Infof("websocket client fd[%d] id[%d] ping", c.fd, c.id)
}

//pong 发送pong包
func (c *websocketProtocol) pong() (iface.IMessage, error) {

	if c.fragmentLength > 125 {
		return nil, util.WebsocketPingPayloadOversize
	}

	message, err := c.nextFrame()
	if err != nil || message == nil {
		return nil, err
	}

	// PING
	firstByte := uint8(10 | 128)
	var encode []byte
	if encode, err = c.encode(firstByte, message.Bytes()); err != nil {
		return nil, err
	}

	// 推送数据
	if _, err := c.push(encode); err != nil {
		return nil, err
	}

	return nil, nil
}

//GetQueryStringParam 获取握手阶段传递过来的参数
func (c *websocketProtocol) GetQueryStringParam() url.Values {
	return c.query
}
