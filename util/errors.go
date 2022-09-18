package util

import "errors"

var HeadBytesLengthFail = errors.New("head bytes fail")
var RouterNotFound = errors.New("router Not Found")
var BodyLenExceedLimit = errors.New("body length exceed limit")
var TLSHandshakeUnFinish = errors.New("tls handshake un finish")
var WebsocketOpcodeFail = errors.New("websocket opcode fail")
var WebsocketRsvFail = errors.New("websocket RSV must be 0")
var WebsocketPingPayloadOversize = errors.New("websocket ping payload oversize")
var WebsocketCtrlMessageMustNotFragmented = errors.New("websocket control message MUST NOT be fragmented")
var WebsocketMustUtf8 = errors.New("websocket text message must utf-8")
