package server

import (
	"fmt"
	"runtime"

	"github.com/ikilobyte/netman/eventloop"

	"github.com/ikilobyte/netman/util"

	"github.com/ikilobyte/netman/iface"
)

type Server struct {
	ip         string
	port       int
	options    *Options              // serve启动可选项参数
	socket     iface.ISocket         // 直接系统调用的方式监听TCP端口，不使用官方的net包
	eventloop  iface.IEventLoop      // 事件循环管理
	connectMgr iface.IConnectManager // 所有的连接管理
	packer     iface.IPacker         // 负责封包解包
	messageCh  chan iface.IMessage   // 负责将消息转发出去的
}

//New 创建Server
func New(ip string, port int, opts ...Option) *Server {

	options := parseOption(opts...)

	// 使用几个epoll
	if options.NumEventLoop <= 0 {
		options.NumEventLoop = 2
	}

	// 处理业务逻辑的goroutine数量，TODO 这个待确定是否需要这个
	if options.NumWorker == 0 {
		options.NumWorker = runtime.NumCPU() * 2
	}

	// 封包解包的实现层
	if options.Packer == nil {
		options.Packer = util.NewDataPacker()
	}

	// 初始化
	server := &Server{
		ip:         ip,
		port:       port,
		options:    options,
		socket:     NewSocket(ip, port),
		eventloop:  eventloop.NewEventLoop(options.NumEventLoop),
		connectMgr: NewConnectManager(),
		messageCh:  make(chan iface.IMessage, 100),
		packer:     options.Packer,
	}

	// 初始化epoll
	server.eventloop.Init()

	// 开启epoll_wait
	server.eventloop.Start()

	// 接收消息的处理，
	go func() {
		for {
			select {
			case msg := <-server.messageCh:
				fmt.Println(msg.GetMsgID())
			}
		}
	}()

	return server
}

//Start 启动
func (s *Server) Start() {

	for {
		conn, err := s.socket.Accept()
		if err != nil {
			fmt.Println("err", err)
			continue
		}

		// 添加到epoll中
		if err := s.eventloop.AddRead(conn); err != nil {
			// 断开连接
			conn.Close()
			continue
		}

		// 添加到统一管理
		s.connectMgr.Add(conn)
	}
}

//Stop 停止
func (s *Server) Stop() {
	fmt.Println("Server.stop")
}

func (s *Server) GetConnectMgr() iface.IConnectManager {
	return s.connectMgr
}

func (s *Server) GetPacker() iface.IPacker {
	return s.packer
}
