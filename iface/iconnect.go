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
	SetEpFd(epfd int)
	GetEpFd() int
	SetPoller(poller IPoller)
	GetPoller() IPoller
	SetWriteBuff([]byte)
	GetWriteBuff() ([]byte, bool)
	SetState(state common.ConnectState) // 外部请勿调用
	SetLastMessageTime(lastMessageTime time.Time)
	GetLastMessageTime() time.Time
	GetTLSEnable() bool
	GetHandshakeCompleted() bool
	SetHandshakeCompleted()
	GetCertificate() tls.Certificate
	GetTLSLayer() *tls.Conn
	GetConnectMgr() IConnectManager
}
