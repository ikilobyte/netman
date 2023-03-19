package iface

import (
	"crypto/tls"
	"net"
	"net/url"
	"time"

	"github.com/ikilobyte/netman/common"
)

type IConnect interface {
	Read(bs []byte) (int, error)
	GetFd() int
	GetID() int
	Close() error
	GetPacker() IPacker
	Send(msgID uint32, bs []byte) (int, error)
	GetAddress() net.Addr
	GetEpFd() int
	GetPoller() IPoller
	GetWriteBuff() ([]byte, bool)
	SetLastMessageTime(lastMessageTime time.Time)
	GetLastMessageTime() time.Time
	GetTLSEnable() bool
	GetHandshakeCompleted() bool
	SetHandshakeCompleted()
	GetCertificate() tls.Certificate
	GetTLSLayer() *tls.Conn
	GetConnectMgr() IConnectManager
	Text([]byte) (int, error)        // 发送websocket text数据
	Binary([]byte) (int, error)      // 发送 websocket 二进制格式数据
	GetQueryStringParam() url.Values // 仅在websocket时可用
	IsUDP() bool
}

//IConnectEvent 专门处理epoll/kqueue事件的方法，无需对外提供
type IConnectEvent interface {
	DecodePacket() (IMessage, error)
	ProceedWrite() error
	SetState(state common.ConnectState)
	SetWriteBuff([]byte)
	SetEpFd(epfd int)
	SetPoller(poller IPoller)
}

type IWebsocketCloser interface {
	CloseCode(code uint16, reason string) error
}
