package server

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"regexp"
	"strings"

	"golang.org/x/sys/unix"

	"github.com/ikilobyte/netman/util"

	"github.com/ikilobyte/netman/iface"
)

const (
	CONTINUATION = iota
	TEXTMODE
	BINMODE
	CLOSE = 8
	PING  = 9
	PONG  = 10
)

type websocketProtocol struct {
	*BaseConnect
	isHandleShake  bool          // 是否已完成握手
	final          uint8         // 本此分帧是否为已完成的包
	fragmentLength uint          // 当前分帧长度
	packetBuffer   *bytes.Buffer // 存储一个完整的数据包
	rBuffer        *bytes.Buffer // 读buffer，保存当前的分帧数据
	parseHeader    bool          // 是否解析了头部，因为是非阻塞模式可能一个分帧会分多次读取
	opcode         uint8         // opcode 操作码
	masks          []byte        // 掩码
	msgID          uint32        // 消息ID
	closeStep      uint8         // 关闭帧步骤
	sendCloseFrame bool
}

//newWebsocketProtocol
func newWebsocketProtocol(baseConnect *BaseConnect) iface.IConnect {

	c := &websocketProtocol{
		BaseConnect:    baseConnect,
		isHandleShake:  false,
		final:          0,
		fragmentLength: 0,
		rBuffer:        bytes.NewBuffer([]byte{}),
		msgID:          0,
		packetBuffer:   bytes.NewBuffer([]byte{}),
		sendCloseFrame: true,
	}

	return c
}

//DecodePacket 读取一个完整的数据包
func (c *websocketProtocol) DecodePacket() (iface.IMessage, error) {

	// 握手
	if c.isHandleShake == false {
		if err := c.handleShake(); err != nil {
			return nil, io.EOF
		}
		c.isHandleShake = true
		// onopen
		c.options.WebsocketHandler.Open(c)
		return nil, nil
	}

	// 解析头部协议
	if c.parseHeader == false {
		headBytes := make([]byte, 2)
		n, err := c.readData(headBytes)
		if n <= 0 || err != nil {
			return nil, err
		}

		// opcode、masks、length等数据
		if err := c.parseHeadBytes(headBytes); err != nil {
			return nil, io.EOF
		}
	}

	// 处理opcode
	switch c.opcode {
	case CONTINUATION:
	case TEXTMODE:
	case BINMODE:
		break
	case CLOSE: // 收到断开连接请求，回复close帧后，等待对方发起fin包
		_ = c.Close()
		return nil, nil
	case PING:
		c.pong() // 需要响应pong
		return nil, nil
	case PONG:
		c.reset()
		return nil, nil
	default:
		return nil, util.WebsocketOpcodeFail
	}

	// 解析payload
	rLen := c.fragmentLength - uint(c.rBuffer.Len())
	payloadBuffer := make([]byte, rLen)
	n, err := c.readData(payloadBuffer)
	if n <= 0 || err != nil {
		return nil, err
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
				Opcode:      c.opcode,
				IsWebSocket: true,
			}

			// 继续重置状态
			c.msgID += 1
			c.packetBuffer = bytes.NewBuffer([]byte{})
			return message, nil
		}
	}

	return nil, nil
}

func (c *websocketProtocol) parseHeadBytes(bs []byte) error {
	firstByte := bs[0]
	secondByte := bs[1]
	c.final = firstByte >> 7 // 当前分帧是否为最后一个包
	c.opcode = firstByte & 0xf
	maskd := secondByte >> 7
	c.fragmentLength = uint(secondByte & 127)

	// 处理payload的长度
	if err := c.parsePayloadLength(); err != nil {
		return err
	}

	// 客户端有做掩码操作，需要继续读取4个字节读取掩码的key，用于解码
	if maskd >= 1 {
		masks := make([]byte, 4)
		n, err := c.readData(masks)
		if n != 4 || err != nil {
			return err
		}
		c.masks = masks
	}

	// 解析头部协议完成
	c.parseHeader = true

	return nil
}

func (c *websocketProtocol) parsePayloadLength() error {

	// 无需解析
	if c.fragmentLength <= 125 {
		return nil
	}

	// 继续读取2个字节表示长度
	if c.fragmentLength == 126 {
		lengthBytes := make([]byte, 2)
		if n, err := c.readData(lengthBytes); n != 2 || err != nil {
			return err
		}
		c.fragmentLength = uint(binary.BigEndian.Uint16(lengthBytes))
		return nil
	}

	// 继续读取8个字节获取长度
	if c.fragmentLength == 127 {
		lengthBytes := make([]byte, 8)
		if n, err := c.readData(lengthBytes); n != 8 || err != nil {
			return err
		}

		c.fragmentLength = uint(binary.BigEndian.Uint64(lengthBytes))
	}

	return nil
}

func (c *websocketProtocol) reset() {
	c.parseHeader = false                 // 是否解析过头部
	c.rBuffer = bytes.NewBuffer([]byte{}) // 分帧buffer
	c.masks = []byte{}                    // 掩码
	c.opcode = 0                          // 操作码
	c.fragmentLength = 0                  // 分帧长度
}

//handleShake websocket握手
func (c *websocketProtocol) handleShake() error {

	buffer := make([]byte, 2048)
	n, err := c.readData(buffer)

	// 连接异常，无需处理
	if n == 0 {
		return err
	}

	// 读取数据异常
	if err != nil {
		util.Logger.Errorf("websocket handle shake err：%v", err)
		return err
	}

	sBuffer := string(buffer)

	// 头部校验
	if strings.Index(sBuffer, "GET / HTTP/1.1") != 0 {
		util.Logger.Errorf("websocket handle shake protocol err：%v", err)
		return io.EOF
	}

	// 边界校验
	if strings.Index(sBuffer, "Connection: Upgrade") == -1 {
		util.Logger.Errorf("websocket handle shake Upgrade err：%v", err)
		return io.EOF
	}

	// 校验是否有相关key
	matches := regexp.MustCompile(`Sec-WebSocket-Key: (.+)`).FindStringSubmatch(sBuffer)
	if len(matches) != 2 {
		util.Logger.Errorf("websocket handle shake Sec-WebSocket-Key err：%v", err)
		return io.EOF
	}

	// 握手协议
	encodeData := fmt.Sprintf("%s258EAFA5-E914-47DA-95CA-C5AB0DC85B11", strings.Trim(matches[1], "\r\n"))
	hash := sha1.New()
	hash.Write([]byte(encodeData))
	bs := hash.Sum(nil)

	headers := "HTTP/1.1 101 Switching Protocols\r\n"
	headers += "Upgrade: websocket\r\n"
	headers += "Connection: Upgrade\r\n"
	headers += fmt.Sprintf("Sec-WebSocket-Accept: %s\r\n", base64.StdEncoding.EncodeToString(bs))
	headers += "Sec-WebSocket-Version: 13\r\n"
	headers += "\r\n"

	n, err = c.Write([]byte(headers))
	return err
}

func (c *websocketProtocol) push(dataBuff []byte) (int, error) {

	if c.GetTLSEnable() {
		return c.tlsLayer.Write(dataBuff)
	}
	return c.Write(dataBuff)
}

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

//Text 发送纯文本格式数据
func (c *websocketProtocol) Text(bs []byte) (int, error) {

	// 第一个字节
	firstByte := uint8(1 | 128)
	encode, err := c.encode(firstByte, bs)
	if err != nil {
		return 0, err
	}

	return c.push(encode)
}

//Binary 发送二进制格式数据
func (c *websocketProtocol) Binary(bs []byte) (int, error) {
	firstByte := uint8(2 | 128)
	encode, err := c.encode(firstByte, bs)
	if err != nil {
		return 0, err
	}
	return c.push(encode)
}

func (c *websocketProtocol) Close() error {

	// 移除事件监听
	_ = c.GetPoller().Remove(c.fd)

	// 从管理类中移除
	c.GetConnectMgr().Remove(c)

	// 发送close帧，code为1000
	_, _ = c.Write([]byte{136, 2, 3, 232})
	err := unix.Close(c.fd)

	// 关闭成功才执行
	if c.hooks != nil && err == nil {
		c.hooks.OnClose(c) // tcp onclose
	}

	// websocket onclose ，握手成功才执行Close回调
	if c.isHandleShake {
		c.options.WebsocketHandler.Close(c)
	}

	// 重置状态
	c.reset()
	c.packetBuffer = nil

	return err
}

//ping 发送ping包
func (c *websocketProtocol) ping() {
	_, _ = c.Write([]byte{137, 0})
	util.Logger.Infof("websocket client fd[%d] id[%d] ping", c.fd, c.id)
}

//pong 发送pong包
func (c *websocketProtocol) pong() {
	_, _ = c.Write([]byte{138, 0})
	c.reset()
	util.Logger.Infof("websocket client fd[%d] id[%d] pong", c.fd, c.id)
}
