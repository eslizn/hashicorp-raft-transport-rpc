package jsonrpc

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"

	"github.com/hashicorp/raft"
)

type Service struct {
	Transport
}

func (s *Service) process(req interface{}, reader io.Reader) (interface{}, error) {
	respCh := make(chan raft.RPCResponse, 1)
	rpc := raft.RPC{
		Command:  req,
		RespChan: respCh,
		Reader:   reader,
	}
	select {
	case <-s.ctx.Done():
		return nil, context.Canceled
	case s.rpcCh <- rpc:
	}
	select {
	case <-s.ctx.Done():
		return nil, context.Canceled
	case res := <-respCh:
		return res.Response, res.Error
	}
}

func (s *Service) AppendEntries(r *raft.AppendEntriesRequest, w *raft.AppendEntriesResponse) error {
	rsp, err := s.process(r, nil)
	if err != nil {
		return err
	}
	w = rsp.(*raft.AppendEntriesResponse)
	return nil
}

func (s *Service) RequestVote(r *raft.RequestVoteRequest, w *raft.RequestVoteResponse) error {
	rsp, err := s.process(r, nil)
	if err != nil {
		return err
	}
	w = rsp.(*raft.RequestVoteResponse)
	return nil
}

func (s *Service) InstallSnapshot(r *InstallSnapshotRequest, w *raft.InstallSnapshotResponse) error {
	rsp, err := s.process(r.InstallSnapshotRequest, bytes.NewReader(r.Data))
	if err != nil {
		return err
	}
	w = rsp.(*raft.InstallSnapshotResponse)
	return nil
}

func (s *Service) TimeoutNow(r *raft.TimeoutNowRequest, w *raft.TimeoutNowResponse) error {
	rsp, err := s.process(r, nil)
	if err != nil {
		return err
	}
	w = rsp.(*raft.TimeoutNowResponse)
	return nil
}

func (s *Service) ServerHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "application/json")
	service := rpc.NewServer()
	if err := service.Register(s); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := service.ServeRequest(jsonrpc.NewServerCodec(&server{ReadCloser: r.Body, Writer: w})); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
