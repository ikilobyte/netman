package util

import "errors"

var HeadBytesLengthFail = errors.New("head bytes fail")
var RouterNotFound = errors.New("router Not Found")
var BodyLenExceedLimit = errors.New("body length exceed limit")
