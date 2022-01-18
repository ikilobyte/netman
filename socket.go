package netman

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/ikilobyte/netman/iface"

	"golang.org/x/sys/unix"
)

type socket struct {
	fd      int
	address string
	nextId  int
}

//GetFd 获取fd
func (s *socket) GetFd() int {
	return s.fd
}

func newSocket(ip string, port int) iface.ISocket {

	socket := &socket{
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
func (s *socket) MakeFd() {
	ListenFd, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, unix.IPPROTO_TCP)
	if err != nil {
		log.Panicln("create socket err", err)
	}

	s.fd = ListenFd
}

//Bind 绑定端口
func (s *socket) Bind() (err error) {

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
func (s *socket) Listen() error {
	return unix.Listen(s.fd, maxListenerBacklog())
}

//Accept 处理新连接
func (s *socket) Accept() (iface.IConnection, error) {

	connFd, sa, err := unix.Accept(s.fd)
	if err != nil {
		return nil, err
	}

	// 设置为不阻塞
	if err := unix.SetNonblock(connFd, true); err != nil {
		return nil, err
	}

	// 返回连接的抽象实例
	conn := newConnection(s.nextId, connFd, sa)
	s.nextId += 1

	return conn, nil
}

func maxListenerBacklog() int {

	fd, err := os.Open("/proc/sys/net/core/somaxconn")
	if err != nil {
		return unix.SOMAXCONN
	}
	defer fd.Close()

	rd := bufio.NewReader(fd)
	line, err := rd.ReadString('\n')
	if err != nil {
		return unix.SOMAXCONN
	}

	f := strings.Fields(line)
	if len(f) < 1 {
		return unix.SOMAXCONN
	}

	n, err := strconv.Atoi(f[0])
	if err != nil || n == 0 {
		return unix.SOMAXCONN
	}
	if n > 1<<16-1 {
		n = 1<<16 - 1
	}
	return n
}
