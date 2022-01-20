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
	emitCh     chan iface.IRequest   // 从这里接收epoll转发过来的消息，然后交给worker去处理
	routerMgr  *RouterMgr            // 路由统一管理
}

//New 创建Server
func New(ip string, port int, opts ...Option) *Server {

	options := parseOption(opts...)

	// 使用几个事件循环管理连接
	if options.NumEventLoop <= 0 {
		options.NumEventLoop = runtime.NumCPU()
	}

	// 封包解包的实现层，外部可以自行实现IPacker使用自己的封包解包方式
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
		emitCh:     make(chan iface.IRequest, 128),
		packer:     options.Packer,
		routerMgr:  NewRouterMgr(),
	}

	// 初始化epoll
	server.eventloop.Init(server.connectMgr)

	// 开启epoll_wait
	server.eventloop.Start(server.emitCh)

	// 接收消息的处理，
	go func() {
		for {
			select {
			case request, ok := <-server.emitCh:

				// 通道已关闭
				if !ok {
					return
				}

				// 交给路由管理中心去处理，执行业务逻辑
				if err := server.routerMgr.Do(request); err != nil {
					fmt.Println("do handler err", err)
				}
			}
		}
	}()

	return server
}

//AddRouter 添加路由处理
func (s *Server) AddRouter(msgID uint32, router iface.IRouter) {
	s.routerMgr.Add(msgID, router)
}

//Start 启动
func (s *Server) Start() {

	fmt.Printf("Server Started %s:%d\n", s.ip, s.port)
	for {
		conn, err := s.socket.Accept(s.packer)
		if err != nil {
			fmt.Println("err", err)
			continue
		}

		// 添加到epoll中
		if err := s.eventloop.AddRead(conn); err != nil {
			// 断开连接
			_ = conn.Close()
			continue
		}

		// 添加到统一管理
		s.connectMgr.Add(conn)

		fmt.Printf("NewConn ID[%d]\n", conn.GetID())
	}
}

//Stop 停止
func (s *Server) Stop() {
	// TODO
	fmt.Println("Server.stop")
}
