package rpctrans

import (
	"context"
	"net/rpc"
	"time"

	"github.com/hashicorp/raft"
)

type future struct {
	respCh chan error
	ctx    context.Context
	start  time.Time
	args   *raft.AppendEntriesRequest
	resp   *raft.AppendEntriesResponse
}

func (f *future) Error() error {
	select {
	case <-f.ctx.Done():
		return context.Canceled
	case err := <-f.respCh:
		return err
	}
}

func (f *future) Start() time.Time {
	return f.start
}

func (f *future) Request() *raft.AppendEntriesRequest {
	return f.args
}

func (f *future) Response() *raft.AppendEntriesResponse {
	return f.resp
}

type pipeline struct {
	ctx         context.Context
	client      *rpc.Client
	doneCh      chan raft.AppendFuture
	progressCh  chan *future
	maxPipeline int
}

func (p *pipeline) progress() error {
	defer p.Close()
	for {
		select {
		case <-p.ctx.Done():
			return context.Canceled
		case call := <-p.progressCh:
			call.respCh <- p.client.Call("AppendEntries", call.args, call.resp)
			p.doneCh <- call
		}
	}
}

func (p *pipeline) AppendEntries(
	args *raft.AppendEntriesRequest,
	resp *raft.AppendEntriesResponse,
) (raft.AppendFuture, error) {
	respCh := make(chan error, 1)
	call := &future{
		ctx:    p.ctx,
		start:  time.Now(),
		args:   args,
		resp:   resp,
		respCh: respCh,
	}
	select {
	case <-p.ctx.Done():
		return nil, context.Canceled
	case p.progressCh <- call:
	}
	return call, nil
}

func (p *pipeline) Consumer() <-chan raft.AppendFuture {
	return p.doneCh
}

func (p *pipeline) Close() error {
	//close(p.doneCh)
	//close(p.progressCh)
	return nil
}
