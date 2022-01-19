package server

import "github.com/ikilobyte/netman/iface"

//Options 可选项配置，未配置时使用默认值
type Options struct {
	NumEventLoop int           // 配置event-loop数量，默认：2
	NumWorker    int           // 用来处理业务逻辑的goroutine数量，默认CPU核心数
	LogPath      string        // 日志路径，默认：logs/${day}.log
	Packer       iface.IPacker // 实现这个接口可以使用自定义的封包方式
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

//WithNumWorker worker goroutine配置，默认CPU核心数量
func WithNumWorker(numWorker int) Option {
	return func(opts *Options) {
		opts.NumWorker = numWorker
	}
}

//WithLogPath 日志保存目录，默认按天保存在logs目录
func WithLogPath(logPath string) Option {
	return func(opts *Options) {
		opts.LogPath = logPath
	}
}

//WithPacker 使用自定义的封包方式
func WithPacker(packer iface.IPacker) Option {
	return func(opts *Options) {
		opts.Packer = packer
	}
}
