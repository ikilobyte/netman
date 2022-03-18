package iface

//IWebsocketHandler websocket协议相关回调
type IWebsocketHandler interface {
	Open(connect IConnect)    // onopen
	Message(request IRequest) // onmessage
	Close(connect IConnect)   // onclose
}
