package server

import (
	"fmt"
	"github.com/ikilobyte/netman/common"
	"github.com/ikilobyte/netman/eventloop"
	"github.com/ikilobyte/netman/iface"
	"github.com/ikilobyte/netman/util"
	"golang.org/x/sys/unix"
	"log"
	"net"
	"runtime"
)

//createUDPServer 初始化udp server
func createUDPServer(ip string, port int, opts ...Option) (*Server, *Options) {

	options := parseOption(opts...)

	// 使用几个事件循环管理连接
	if options.NumEventLoop <= 0 {
		options.NumEventLoop = runtime.NumCPU()
	}

	// 封包解包的实现层，外部可以自行实现IPacker使用自己的封包解包方式
	if options.Packer == nil {
		options.Packer = util.NewDataPacker()
		options.Packer.SetMaxBodyLength(options.MaxBodyLength)
	}

	if options.UDPPacketBufferLength <= 0 {
		options.UDPPacketBufferLength = 32768
	}

	// 日志保存路径
	if options.LogOutput != nil {
		util.Logger.SetOutput(options.LogOutput)
	}

	// 初始化
	server := &Server{
		ip:         ip,
		port:       port,
		network:    "udp",
		options:    options,
		status:     stopped,
		socket:     newUdpSocket(ip, port),
		eventloop:  eventloop.NewEventLoop(options.NumEventLoop),
		connectMgr: newConnectManager(options),
		emitCh:     make(chan iface.IContext, 128),
		packer:     options.Packer,
		routerMgr:  NewRouterMgr(),
	}

	// 初始化epoll
	//if err := server.eventloop.Init(server.connectMgr); err != nil {
	//	log.Panicln(err)
	//}
	// 执行wait
	//server.eventloop.Start(server.emitCh)

	server.acceptor = newAcceptorUdp(
		server.packer,
		server.connectMgr,
		options,
	)

	// 处理消息
	go server.doMessage()

	return server, options
}

//newUdpSocket 创建一个udp socket
func newUdpSocket(ip string, port int) *socket {

	// 创建一个UDP socket
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM|unix.SOCK_CLOEXEC, unix.IPPROTO_UDP)
	if err != nil {
		log.Panicln(err)
	}

	udpAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		log.Panicln(err)
	}

	if err := unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEPORT, 1); err != nil {
		log.Panicln(err)
	}

	if err := unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1); err != nil {
		log.Panicln(err)
	}
	v4 := udpAddr.IP.To4()
	// 端口绑定
	err = unix.Bind(fd, &unix.SockaddrInet4{
		Port: port,
		Addr: [4]byte{
			v4[0],
			v4[1],
			v4[2],
			v4[3],
		},
	})
	if err != nil {
		log.Panicln(err)
	}

	return &socket{
		fd: fd,
		//socketId: -1,
	}
}

func UDP(ip string, port int, opts ...Option) *Server {
	server, options := createUDPServer(ip, port, opts...)
	options.Application = common.RouterMode
	return server
}
