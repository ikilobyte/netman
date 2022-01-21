package server

import (
	"io"
	"time"

	"github.com/ikilobyte/netman/iface"
)

//Options 可选项配置，未配置时使用默认值
type Options struct {
	NumEventLoop int           // 配置event-loop数量，默认：2
	NumWorker    int           // 用来处理业务逻辑的goroutine数量，默认CPU核心数
	LogOutput    io.Writer     // 日志保存目标，默认：Stdout
	Packer       iface.IPacker // 实现这个接口可以使用自定义的封包方式
	TCPKeepAlive time.Duration // TCP keepalive
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
