package jsonrpc

import (
	"bytes"
	"errors"
	"net/http"
)

type client struct {
	url string
	req *http.Request
	rsp *http.Response
}

func (c *client) Read(p []byte) (int, error) {
	if c.rsp == nil {
		return 0, errors.New("response is nil")
	}
	return c.rsp.Body.Read(p)
}

func (c *client) Write(p []byte) (int, error) {
	if c.req != nil {
		return 0, errors.New("request is not nil")
	}
	var (
		err error
	)
	c.req, err = http.NewRequest("POST", c.url, bytes.NewBuffer(p))
	if err != nil {
		return 0, err
	}
	c.req.Header.Set("Content-type", "application/json")
	c.rsp, err = http.DefaultClient.Do(c.req)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func (c *client) Close() error {
	if c.req != nil {
		c.req.Body.Close()
		c.req = nil
	}
	if c.rsp != nil {
		c.rsp.Body.Close()
		c.rsp = nil
	}
	return nil
}
