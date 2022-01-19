package util

import "errors"

var ConnectClosed = errors.New("connect is closed")
var HeadBytesLengthFail = errors.New("head bytes fail")
