package netman

import (
	"fmt"
	"log"
	"runtime"

	"github.com/ikilobyte/netman/iface"
)

type Server struct {
	ip        string
	port      int
	options   *Options
	socket    iface.ISocket
	poller    []iface.IPoller
	connMgr   iface.IConnManager
	messageCh chan message
}

func (s *Server) Emit(dataBuff []byte) {
	msg := newMessage(dataBuff)
	s.messageCh <- *msg
}

func (s *Server) GetConnMgr() iface.IConnManager {
	return s.connMgr
}

//NewServer 创建Server
func NewServer(ip string, port int, opts ...Option) *Server {

	options := parseOption(opts...)

	// 默认2个
	if options.NumEventLoop <= 0 {
		options.NumEventLoop = 2
	}

	if options.NumWorker <= 0 {
		options.NumWorker = runtime.NumCPU()
	}

	server := &Server{
		ip:        ip,
		port:      port,
		options:   options,
		socket:    newSocket(ip, port),
		poller:    make([]iface.IPoller, options.NumEventLoop),
		connMgr:   newConnManager(),
		messageCh: make(chan message, 100),
	}

	// 开启 epoll
	server.startPoller()

	// 开启 worker
	server.startWorker()

	return server
}

//Start 启动服务
func (s *Server) Start() {

	// 单acceptor 多event-loop，多worker、模型
	for {
		// 接收新过来的连接
		conn, err := s.socket.Accept()
		if err != nil {
			fmt.Println("err", err)
			continue
		}

		// 获取一个poller，添加fd到事件循环中
		poller := s.getPoller(conn)
		if err := poller.AddRead(conn.GetFd(), conn.GetID()); err != nil {
			fmt.Println("poller.AddRead.err", err)
			continue
		}

		// 将这个连接放到连接管理中
		fmt.Println("Len -> ", s.connMgr.Add(conn))
	}
}

func (s *Server) Stop() {
	fmt.Println("Server.stop")
}

func (s *Server) startPoller() {
	for i := 0; i < s.options.NumEventLoop; i++ {
		poller, err := newPoller(s)

		// 创建不成功直接panic
		if err != nil {
			log.Panicln(err)
		}
		s.poller[i] = poller

		// event wait
		go poller.Wait()
	}
}

func (s *Server) startWorker() {
	for i := 0; i < s.options.NumWorker; i++ {
		worker := newWorker(i, s.messageCh)
		go worker.Start()
		fmt.Printf("worker-%d started\n", i)
	}
}

func (s *Server) getPoller(conn iface.IConnection) iface.IPoller {
	idx := conn.GetID() % s.options.NumEventLoop
	return s.poller[idx]
}
