package iface

import (
	"crypto/tls"
	"net"
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
}

//IConnectEvent 专门处理epoll/kqueue事件的方法，无需对外提供
type IConnectEvent interface {
	ProceedWrite() error
	SetState(state common.ConnectState)
	SetWriteBuff([]byte)
	SetEpFd(epfd int)
	SetPoller(poller IPoller)
}
