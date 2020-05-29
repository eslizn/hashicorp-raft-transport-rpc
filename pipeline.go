package jsonrpc

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
	ctxCancel   context.CancelFunc
	client      *rpc.Client
	doneCh      chan raft.AppendFuture
	progressCh  chan *future
	maxPipeline int
}

func (p *pipeline) progress() (retErr error) {
	respCh := make(chan *future, p.maxPipeline)
	defer func() {
		p.Close()
		close(respCh)
		for call := range respCh {
			call.respCh <- retErr
		}
	}()
	//p.client.Call()
	//p.client.Go()
	//go func() {
	//	defer p.Close()
	//	for {
	//		//p.client.Call()
	//
	//		msg, err := p.client.Recv()
	//		if err != nil {
	//			return
	//		}
	//		select {
	//		case <-p.ctx.Done():
	//			return
	//		case r := <-respCh:
	//			if msg.Error != "" {
	//				err := errors.New(msg.Error)
	//				r.respCh <- err
	//				return
	//			}
	//
	//			msg.Response.CopyToRaft(r.resp)
	//			r.respCh <- nil
	//			p.doneCh <- r
	//		}
	//	}
	//}()
	//
	//for {
	//	select {
	//	case <-p.ctx.Done():
	//		return context.Canceled
	//	case call := <-p.pipelineCh:
	//		select {
	//		case respCh <- call:
	//		case <-p.ctx.Done():
	//			call.respCh <- context.Canceled
	//			return context.Canceled
	//		}
	//		p.client.Call(call.args.Term)
	//		if err := p.client.Send(call.args); err != nil {
	//			return err
	//		}
	//	}
	//}
	return nil
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
	p.ctxCancel()
	return nil
}
