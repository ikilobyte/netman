package server

import (
	"log"
	"net"
	"syscall"
	"time"

	"github.com/ikilobyte/netman/util"

	"github.com/ikilobyte/netman/iface"

	"golang.org/x/sys/unix"
)

type socket struct {
	fd       int
	socketId int
}

//newSocket 使用系统调用创建socket，不使用net包，net包未暴露fd的相关接口，只能通过反射获取，效率不高
func createSocket(address string, duration time.Duration) *socket {

	// 创建
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, unix.IPPROTO_TCP)
	if err != nil {
		util.Logger.Errorf("socket create error %v", err)
		panic(err)
	}

	// 设置属性
	if secs := duration / time.Second; secs >= 1 {
		if err := setKeepAlive(fd, int(secs)); err != nil {
			panic(err)
		}
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

	if err := unix.SetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_KEEPINTVL, secs); err != nil {
		return err
	}

	return unix.SetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_KEEPIDLE, secs)
}

//Accept 处理新连接
func (s *socket) Accept(packer iface.IPacker) (iface.IConnect, error) {

	connFd, sa, err := unix.Accept(s.fd)
	if err != nil {
		return nil, err
	}

	// 设置非阻塞
	if err := unix.SetNonblock(connFd, true); err != nil {
		return nil, err
	}

	// 设置不延迟
	if err := unix.SetsockoptInt(connFd, syscall.IPPROTO_TCP, syscall.TCP_NODELAY, 1); err != nil {
		return nil, err
	}

	// 创建一个连接实例
	conn := newConnect(s.incrementID(), connFd, util.SockaddrToTCPOrUnixAddr(sa), packer)
	return conn, nil
}

func (s *socket) incrementID() int {
	s.socketId += 1
	return s.socketId
}
