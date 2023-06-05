package server

import (
	"fmt"
	"log"
	"runtime"

	"github.com/ikilobyte/netman/common"
	"github.com/ikilobyte/netman/eventloop"
	"github.com/ikilobyte/netman/iface"
	"github.com/ikilobyte/netman/util"
	"golang.org/x/sys/unix"
)

type serverStatus = int

const (
	stopped  serverStatus = iota // 已停止（初始状态）
	started                      // 已启动
	stopping                     // 停止中
)

type Server struct {
	ip         string
	port       int
	network    string                // tcp还是udp
	status     serverStatus          // 状态
	options    *Options              // serve启动可选项参数
	socket     *socket               // 直接系统调用的方式监听TCP端口，不使用官方的net包
	acceptor   iface.IAcceptor       // 处理新连接
	eventloop  iface.IEventLoop      // 事件循环管理
	connectMgr iface.IConnectManager // 所有的连接管理
	packer     iface.IPacker         // 负责封包解包
	emitCh     chan iface.IContext   // 从这里接收epoll转发过来的消息，然后交给worker去处理
	routerMgr  *RouterMgr            // 路由统一管理
}

// makeServer 创建tcp server服务器
func createTcpServer(ip string, port int, opts ...Option) (*Server, *Options) {

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

	// 日志保存路径
	if options.LogOutput != nil {
		util.Logger.SetOutput(options.LogOutput)
	}

	// 初始化
	server := &Server{
		ip:         ip,
		port:       port,
		network:    "tcp",
		options:    options,
		status:     stopped,
		socket:     createSocket(fmt.Sprintf("%s:%d", ip, port), options.TCPKeepAlive),
		eventloop:  eventloop.NewEventLoop(options.NumEventLoop),
		connectMgr: newConnectManager(options),
		emitCh:     make(chan iface.IContext, 128),
		packer:     options.Packer,
		routerMgr:  NewRouterMgr(),
	}

	// 初始化epoll
	if err := server.eventloop.Init(server.connectMgr); err != nil {
		log.Panicln(err)
	}

	// 执行wait
	server.eventloop.Start(server.emitCh)
	server.acceptor = newAcceptor(
		server.packer,
		server.connectMgr,
		options,
	)

	// 处理消息
	go server.doMessage()

	return server, options
}

// New 创建Server
func New(ip string, port int, opts ...Option) *Server {

	server, options := createTcpServer(ip, port, opts...)

	// 应用层协议模式
	options.Application = common.RouterMode

	return server
}

// Websocket 创建一个websocket server
func Websocket(ip string, port int, handler iface.IWebsocketHandler, opts ...Option) *Server {
	server, options := createTcpServer(ip, port, opts...)

	// 应用层协议模式
	options.Application = common.WebsocketMode
	options.WebsocketHandler = handler

	return server
}

// AddRouter 添加路由处理
func (s *Server) AddRouter(msgID uint32, router iface.IRouter) {

	// 只有路由模式才可以添加
	if s.options.Application != common.RouterMode {
		log.Panicln("application not websocket")
		return
	}

	s.routerMgr.Add(msgID, router)
}

// Start 启动
func (s *Server) Start() {
	if s.status != stopped {
		return
	}
	s.status = started

	// 处理路由分组的数据
	if err := s.routerMgr.ResolveGroup(); err != nil {
		util.Logger.Errorf("server start error：%v", err)
	}

	if err := s.acceptor.Run(s.socket.fd, s.eventloop); err != nil {
		util.Logger.Errorf("server start error：%v", err)
	}
}

// doMessage 处理消息
func (s *Server) doMessage() {
	for {
		select {
		case context, ok := <-s.emitCh:

			// 通道已关闭
			if !ok {
				return
			}

			// 分发出去
			go s.routerMgr.Dispatch(context, s.options)
		}
	}
}

// Use 全局中间件
func (s *Server) Use(callable iface.MiddlewareFunc) *Server {
	s.routerMgr.globalMiddlewares = append(s.routerMgr.globalMiddlewares, callable)
	return s
}

// Group 分组中间件
func (s *Server) Group(callable iface.MiddlewareFunc, more ...iface.MiddlewareFunc) iface.IMiddlewareGroup {
	return s.routerMgr.NewGroup(callable, more...)
}

// Stop 停止
func (s *Server) Stop() {
	s.status = stopping
	s.connectMgr.ClearAll()
	s.eventloop.Stop()
	close(s.emitCh)
	_ = unix.Close(s.socket.fd)
	s.acceptor.Exit()
}
