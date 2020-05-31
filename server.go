package jsonrpc

import "io"

type server struct {
	io.ReadCloser
	io.Writer
}
