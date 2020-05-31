package jsonrpc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/raft"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
)

type InstallSnapshotRequest struct {
	*raft.InstallSnapshotRequest
	Data []byte
}

type rpcServer struct {
	io.ReadCloser
	io.Writer
}

type rpcClient struct {
	ctx  context.Context
	url  *url.URL
	data chan io.ReadCloser
}

func (r *rpcClient) Read(buff []byte) (int, error) {
	select {
	case <-r.ctx.Done():
		return 0, context.Canceled
	case data := <-r.data:
		defer data.Close()
		var err error
		buff, err = ioutil.ReadAll(data)
		fmt.Printf("rsp:%s\n", string(buff))
		if err != nil {
			return 0, err
		}
		return len(buff), nil
	}
}

func (r *rpcClient) Write(buff []byte) (int, error) {
	fmt.Printf("Req(%s): %s\n", r.url.String(), string(buff))
	req, err := http.NewRequestWithContext(r.ctx, "POST", r.url.String(), bytes.NewBuffer(buff))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-type", "application/json")
	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	if rsp.StatusCode != http.StatusOK {
		return 0, errors.New(rsp.Status)
	}
	select {
	case <-r.ctx.Done():
		return 0, context.Canceled
	case r.data <- rsp.Body:
		return len(buff), nil
	}
}

func (r *rpcClient) Close() error {
	return nil
}
