package util

import (
	"github.com/ikilobyte/netman/iface"
)

type Pipeline struct {
	passable interface{}
	pipes    []iface.IStage
}

func NewPipeline() iface.IPipeline {
	return &Pipeline{
		pipes: make([]iface.IStage, 0),
	}
}

//Send 需要处理的数据
func (p *Pipeline) Send(passable interface{}) iface.IPipeline {
	p.passable = passable
	return p
}

//Pipe 单个管道
func (p *Pipeline) Pipe(pipe iface.IStage) iface.IPipeline {
	p.pipes = append(p.pipes, pipe)
	return p
}

//Through 一次增加多个管道
func (p *Pipeline) Through(pipes []iface.IStage) iface.IPipeline {
	p.pipes = append(p.pipes, pipes...)
	return p
}

//Then 将最终的结果输出到 destination
func (p *Pipeline) Then(destination iface.NextFunc) interface{} {

	pipes := make([]iface.IStage, 0)
	for i := len(p.pipes) - 1; i >= 0; i-- {
		pipes = append(pipes, p.pipes[i])
	}

	pipeline := ArrayReduce(pipes, p.Carry(), destination)

	// 执行结果
	return pipeline.(iface.PipeFunc)(p.passable)
}

func (p *Pipeline) Carry() iface.CarryFunc {

	return func(stack interface{}, stage interface{}) interface{} {

		return func(passable interface{}) interface{} {
			return stage.(iface.IStage).Process(passable, stack.(iface.NextFunc))
		}
	}
}
