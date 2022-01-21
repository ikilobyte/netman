package server

import (
	"fmt"
	"log"
	"net"
	"syscall"

	"github.com/ikilobyte/netman/util"

	"github.com/ikilobyte/netman/iface"

	"golang.org/x/sys/unix"
)

type Socket struct {
	fd      int
	address string
	nextId  int
}

//GetFd 获取fd
func (s *Socket) GetFd() int {
	return s.fd
}

func NewSocket(ip string, port int) iface.ISocket {

	socket := &Socket{
		fd:      0,
		address: fmt.Sprintf("%s:%d", ip, port),
	}

	// 创建fd
	socket.MakeFd()

	// 绑定
	if err := socket.Bind(); err != nil {
		log.Panicln(err)
	}

	// 监听端口
	if err := socket.Listen(); err != nil {
		log.Panicln(err)
	}

	return socket
}

//MakeFd 创建描述符
func (s *Socket) MakeFd() {
	ListenFd, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, unix.IPPROTO_TCP)
	if err != nil {
		log.Panicln("create socket err", err)
	}

	s.fd = ListenFd
}

//Bind 绑定端口
func (s *Socket) Bind() (err error) {

	tcpAddr, err := net.ResolveTCPAddr("tcp", s.address)
	if err != nil {
		return err
	}

	sa := &unix.SockaddrInet4{Port: tcpAddr.Port}

	// 绑定的端口
	err = unix.Bind(s.fd, sa)
	if err != nil {
		fmt.Println("socket bind err", err)
		return err
	}

	return nil
}

//Listen 监听端口
func (s *Socket) Listen() error {
	return unix.Listen(s.fd, util.MaxListenerBacklog())
}

//Accept 处理新连接
func (s *Socket) Accept(packer iface.IPacker) (iface.IConnect, error) {

	connFd, sa, err := unix.Accept(s.fd)
	if err != nil {
		return nil, err
	}

	// 设置为不阻塞
	if err := unix.SetNonblock(connFd, true); err != nil {
		return nil, err
	}

	if err := unix.SetsockoptInt(connFd, syscall.IPPROTO_TCP, syscall.TCP_NODELAY, 1); err != nil {
		return nil, err
	}

	// 创建一个连接
	conn := NewConnect(
		s.nextId,
		connFd,
		util.SockaddrToTCPOrUnixAddr(sa),
		packer,
	)
	s.nextId += 1

	return conn, nil
}
