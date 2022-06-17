package iface

type IStage interface {
	Process(value interface{}, next NextFunc) interface{}
}
