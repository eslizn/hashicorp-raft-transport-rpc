package jsonrpc

import "errors"

var (
	ErrInvalidType      = errors.New("invalid type")
	ErrResponseNotReady = errors.New("response not ready")
)
