package jsonrpc

import (
	"bytes"
	"context"
	"github.com/hashicorp/raft"
	"github.com/jinzhu/copier"
	"io"
)

type Service Transport

func (s *Service) process(req interface{}, reader io.Reader, isHeartbeat bool) (interface{}, error) {
	respCh := make(chan raft.RPCResponse, 1)
	rpc := raft.RPC{
		Command:  req,
		RespChan: respCh,
		Reader:   reader,
	}
	s.heartbeatFnLock.RLock()
	fn := s.heartbeatFn
	s.heartbeatFnLock.RUnlock()
	if isHeartbeat && fn != nil {
		fn(rpc)
	} else {
		select {
		case <-s.ctx.Done():
			return nil, context.Canceled
		case s.rpcCh <- rpc:
		}
	}

	select {
	case <-s.ctx.Done():
		return nil, context.Canceled
	case res := <-respCh:
		return res.Response, res.Error
	}
}

func (s *Service) AppendEntries(r *raft.AppendEntriesRequest, w *raft.AppendEntriesResponse) error {
	isHeartbeat := r.Term != 0 && r.Leader != nil &&
		r.PrevLogEntry == 0 && r.PrevLogTerm == 0 &&
		len(r.Entries) == 0 && r.LeaderCommitIndex == 0
	rsp, err := s.process(r, nil, isHeartbeat)
	if err != nil {
		return err
	}
	copier.Copy(w, rsp)
	return nil
}

func (s *Service) RequestVote(r *raft.RequestVoteRequest, w *raft.RequestVoteResponse) error {
	rsp, err := s.process(r, nil, false)
	if err != nil {
		return err
	}
	copier.Copy(w, rsp)
	return nil
}

func (s *Service) InstallSnapshot(r *InstallSnapshotRequest, w *raft.InstallSnapshotResponse) error {
	rsp, err := s.process(r.InstallSnapshotRequest, bytes.NewReader(r.Data), false)
	if err != nil {
		return err
	}
	copier.Copy(w, rsp)
	return nil
}

func (s *Service) TimeoutNow(r *raft.TimeoutNowRequest, w *raft.TimeoutNowResponse) error {
	rsp, err := s.process(r, nil, false)
	if err != nil {
		return err
	}
	copier.Copy(w, rsp)
	return nil
}

//func Listen(srv *Service) error {
//	parse, err := url.Parse(string(srv.addr))
//	if err != nil {
//		return err
//	}
//	//listen := &http.Server{
//	//	Addr:    parse.Host,
//	//	Handler: ServeHTTP(srv),
//	//}
//	//errCh := make(chan error)
//	//go func() {
//	//	errCh <- listen.ListenAndServe()
//	//}()
//	server := rpc.NewServer()
//	server.Register(srv)
//	listen, err := net.Listen(parse.Scheme, parse.Host)
//	if err != nil {
//		return err
//	}
//	defer listen.Close()
//	go func() {
//		server.Accept(listen)
//	}()
//	<-srv.ctx.Done()
//	return context.Canceled
//}

func NewService(ctx context.Context, id string, addr string) *Service {
	return &Service{
		id:          raft.ServerID(id),
		addr:        raft.ServerAddress(addr),
		ctx:         ctx,
		rpcCh:       make(chan raft.RPC),
		maxPipeline: 8,
	}
}
