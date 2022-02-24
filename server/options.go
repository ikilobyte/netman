package server

import (
	"io"
	"time"

	"github.com/ikilobyte/netman/iface"
)

//Options 可选项配置，未配置时使用默认值
type Options struct {
	NumEventLoop           int           // 配置event-loop数量，默认：2
	NumWorker              int           // 用来处理业务逻辑的goroutine数量，默认CPU核心数
	LogOutput              io.Writer     // 日志保存目标，默认：Stdout
	Packer                 iface.IPacker // 实现这个接口可以使用自定义的封包方式
	TCPKeepAlive           time.Duration // TCP keepalive
	Hooks                  iface.IHooks  // hooks
	MaxBodyLength          uint32        // 包体部分最大长度，默认：0(不限制大小)
	HeartbeatCheckInterval time.Duration // 表示多久进行轮询一次心跳检测
	HeartbeatIdleTime      time.Duration // 连接最大允许空闲的时间，二者需要同时配置才会生效
}

type Option = func(opts *Options)

//parseOption 解析可选项
func parseOption(opts ...Option) *Options {
	options := new(Options)
	for _, opt := range opts {
		opt(options)
	}

	return options
}

//WithNumEventLoop event-loop数量配置
func WithNumEventLoop(numEventLoop int) Option {
	return func(opts *Options) {
		opts.NumEventLoop = numEventLoop
	}
}

//WithTCPKeepAlive 设置时间 TCP keepalive
func WithTCPKeepAlive(duration time.Duration) Option {
	return func(opts *Options) {
		opts.TCPKeepAlive = duration
	}
}

//WithLogOutput 日志保存目录，默认按天保存在logs目录
func WithLogOutput(output io.Writer) Option {
	return func(opts *Options) {
		opts.LogOutput = output
	}
}

//WithPacker 使用自定义的封包方式
func WithPacker(packer iface.IPacker) Option {
	return func(opts *Options) {
		opts.Packer = packer
	}
}

//WithHooks hooks
func WithHooks(hooks iface.IHooks) Option {
	return func(opts *Options) {
		opts.Hooks = hooks
	}
}

//WithMaxBodyLength 配置包体部分最大长度
func WithMaxBodyLength(length uint32) Option {
	return func(opts *Options) {
		opts.MaxBodyLength = length
	}
}

//WithHeartbeatCheckInterval 服务端多长时间检测一次客户端心跳
func WithHeartbeatCheckInterval(interval time.Duration) Option {
	return func(opts *Options) {
		opts.HeartbeatCheckInterval = interval
	}
}

//WithHeartbeatIdleTime 允许连接最大空闲时间，也就是允许连接最长多少时间不发送消息
func WithHeartbeatIdleTime(idleTime time.Duration) Option {
	return func(opts *Options) {
		opts.HeartbeatIdleTime = idleTime
	}
}
