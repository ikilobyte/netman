package server

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"
	"syscall"

	"github.com/ikilobyte/netman/iface"
	"github.com/ikilobyte/netman/util"
)

const (
	CONTINUATION = iota
	TEXTMODE
	BINMODE
	CLOSE = 8
	PING  = 9
	PONG  = 10
)

type headerStep = uint8

const (
	parsePayloadLength headerStep = iota + 1
	parseMasks
)

type websocketProtocol struct {
	*BaseConnect
	isHandleShake   bool          // 是否已完成握手
	final           uint8         // 本此分帧是否为已完成的包
	fragmentLength  uint          // 当前分帧长度
	packetBuffer    *bytes.Buffer // 存储一个完整的数据包
	rBuffer         *bytes.Buffer // 读buffer，保存当前的分帧数据
	parseHeader     bool          // 是否解析了头部，因为是非阻塞模式可能一个分帧会分多次读取
	opcode          uint8         // opcode 操作码
	masks           []byte        // 掩码
	msgID           uint32        // 消息ID
	closeStep       uint8         // 关闭帧步骤
	sendCloseFrame  bool          //
	query           url.Values    // 在握手阶段传过来的query参数
	messageMode     uint8         // 消息类型
	parseHeaderStep uint8         // 解析头数据到了第几个步骤
	headerBytes     []byte
}

//newWebsocketProtocol
func newWebsocketProtocol(baseConnect *BaseConnect) iface.IConnect {

	c := &websocketProtocol{
		BaseConnect:     baseConnect,
		isHandleShake:   false,
		final:           0,
		fragmentLength:  0,
		rBuffer:         bytes.NewBuffer([]byte{}),
		msgID:           0,
		packetBuffer:    bytes.NewBuffer([]byte{}),
		sendCloseFrame:  true,
		query:           make(url.Values),
		messageMode:     0,
		masks:           []byte{},
		parseHeaderStep: 0,
		headerBytes:     []byte{},
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

		// 读取头2个字节
		if length := 2 - len(c.headerBytes); length > 0 {
			headerBytes := make([]byte, length)
			n, err := c.readData(headerBytes)
			// 优化到外层去处理
			if n <= 0 || err != nil {
				return nil, err
			}

			c.headerBytes = append(c.headerBytes, headerBytes[:n]...)
		}

		if len(c.headerBytes) != 2 {
			return nil, syscall.EAGAIN
		}
		// opcode、masks、length等数据
		if err := c.parseHeadBytes(c.headerBytes); err != nil {
			return nil, err
		}
	}

	// 没有解析完成header，不能继续执行
	if !c.parseHeader {
		return nil, syscall.EAGAIN
	}

	//fmt.Printf(
	//	"parseHeader %v header bytes %v masks %v opcode %d fragmentLength %d final %d\n",
	//	c.parseHeader,
	//	c.headerBytes,
	//	c.masks,
	//	c.opcode,
	//	c.fragmentLength,
	//	c.final,
	//)
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
		return c.pong()
	case PONG:
		// 没有带数据包的PONG，直接关闭
		if c.fragmentLength <= 0 {
			_ = c.Close()
			return nil, nil
		}

		// 读取PONG携带的数据包
		_, err := c.nextFrame()
		return nil, err
	default:
		return nil, util.WebsocketOpcodeFail
	}

	// 读取帧
	return c.nextFrame()
}

func (c *websocketProtocol) parseHeadBytes(bs []byte) error {
	firstByte := bs[0]
	secondByte := bs[1]
	c.final = firstByte >> 7 // 当前分帧是否为最后一个包
	c.opcode = firstByte & 0xf
	maskd := secondByte >> 7
	c.fragmentLength = uint(secondByte & 127)

	// 保存这个分帧的消息类型
	if c.opcode == TEXTMODE || c.opcode == BINMODE {
		c.messageMode = c.opcode
	}

	// 读取数据解析出
	if c.parseHeaderStep < parsePayloadLength {
		if err := c.parsePayloadLength(); err != nil {
			return err
		}
	}

	// 客户端有做掩码操作，需要继续读取4个字节读取掩码的key，用于解码
	if maskd >= 1 {
		if c.parseHeaderStep < parseMasks {
			if length := 4 - len(c.masks); length > 0 {
				masks := make([]byte, length)
				n, err := c.readData(masks)

				if n <= 0 || err != nil {
					return err
				}

				c.masks = append(c.masks, masks[:n]...)
			}

			if len(c.masks) == 4 {
				c.parseHeaderStep = parseMasks // 当前步骤
				c.parseHeader = true           // 标记为已解析
				c.headerBytes = []byte{}       // 重置这个头部的字节数据
			}
		}

	}

	return nil
}

func (c *websocketProtocol) parsePayloadLength() error {

	// 无需解析
	if c.fragmentLength <= 125 {
		c.parseHeaderStep = parsePayloadLength
		return nil
	}

	// 继续读取2个字节表示长度
	if c.fragmentLength == 126 {
		lengthBytes := make([]byte, 2)
		if n, err := c.readData(lengthBytes); n != 2 || err != nil {
			return err
		}
		c.fragmentLength = uint(binary.BigEndian.Uint16(lengthBytes))
		c.parseHeaderStep = parsePayloadLength
		return nil
	}

	// 继续读取8个字节获取长度
	if c.fragmentLength == 127 {
		lengthBytes := make([]byte, 8)
		if n, err := c.readData(lengthBytes); n != 8 || err != nil {
			return err
		}

		c.fragmentLength = uint(binary.BigEndian.Uint64(lengthBytes))
		c.parseHeaderStep = parsePayloadLength
	}

	return nil
}

func (c *websocketProtocol) reset() {
	c.parseHeader = false                 // 是否解析过头部
	c.rBuffer = bytes.NewBuffer([]byte{}) // 分帧buffer
	c.masks = []byte{}                    // 掩码
	c.opcode = 0                          // 操作码
	c.fragmentLength = 0                  // 分帧长度
	c.parseHeaderStep = 0
	c.headerBytes = []byte{}
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
	headMatches := regexp.MustCompile(`GET /(.*?) HTTP/1.1`).FindStringSubmatch(sBuffer)
	if len(headMatches) != 2 {
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

	// 解析query string
	split := strings.Split(headMatches[1], "?")
	if len(split) == 2 {
		c.query, _ = url.ParseQuery(split[1])
	}

	// 响应握手协议
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

//Close 关闭连接
func (c *websocketProtocol) Close() error {
	// 发送close帧，code为1000
	return c.close(1000, "")
}
