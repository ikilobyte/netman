//+build darwin

package server

import (
	"log"
	"net"
	"time"

	"github.com/ikilobyte/netman/util"

	"golang.org/x/sys/unix"
)

type socket struct {
	fd       int
	socketId int
}

//newSocket 使用系统调用创建socket，不使用net包，net包未暴露fd的相关接口，只能通过反射获取，效率不高
func createSocket(address string, duration time.Duration) *socket {

	// 创建
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM, unix.IPPROTO_TCP)
	if err != nil {
		log.Panicln(err)
	}

	// 设置属性
	if secs := int(duration / time.Second); secs >= 1 {
		if err := setKeepAlive(fd, secs); err != nil {
			log.Panicln(err)
		}
	}

	// 复用TIME_WAIT状态的端口
	if err := unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1); err != nil {
		log.Panicln(err)
	}

	// 绑定
	tcpAddr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		log.Panicln(err)
	}

	// 绑定端口
	if err := unix.Bind(fd, &unix.SockaddrInet4{Port: tcpAddr.Port}); err != nil {
		log.Panicln(err)
	}

	// 监听端口
	if err := unix.Listen(fd, util.MaxListenerBacklog()); err != nil {
		log.Panicln(err)
	}

	return &socket{
		fd:       fd,
		socketId: -1,
	}
}

//setKeepAlive 设置tcp属性
func setKeepAlive(fd, secs int) error {
	if secs <= 0 {
		return nil
	}

	if err := unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_KEEPALIVE, 1); err != nil {
		return err
	}

	switch err := unix.SetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_KEEPINTVL, secs); err {
	case nil, unix.ENOPROTOOPT: // OS X 10.7 and earlier don't support this option
	default:
		return err
	}

	return unix.SetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_KEEPALIVE, secs)
}
