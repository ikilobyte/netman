package iface

type NextFunc = func(value interface{}) interface{}
type CarryFunc = func(stack interface{}, item interface{}) interface{}
type PipeFunc = func(passable interface{}) interface{}

type IPipeline interface {
	Send(passable interface{}) IPipeline
	Pipe(pipe IStage) IPipeline
	Through(pipes []IStage) IPipeline
	Then(destination NextFunc) interface{}
	Carry() CarryFunc
}
