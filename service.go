package jsonrpc

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"
	"net/url"

	"github.com/hashicorp/raft"
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
	w = rsp.(*raft.AppendEntriesResponse)
	return nil
}

func (s *Service) RequestVote(r *raft.RequestVoteRequest, w *raft.RequestVoteResponse) error {
	rsp, err := s.process(r, nil, false)
	if err != nil {
		return err
	}
	w = rsp.(*raft.RequestVoteResponse)
	return nil
}

func (s *Service) InstallSnapshot(r *InstallSnapshotRequest, w *raft.InstallSnapshotResponse) error {
	rsp, err := s.process(r.InstallSnapshotRequest, bytes.NewReader(r.Data), false)
	if err != nil {
		return err
	}
	w = rsp.(*raft.InstallSnapshotResponse)
	return nil
}

func (s *Service) TimeoutNow(r *raft.TimeoutNowRequest, w *raft.TimeoutNowResponse) error {
	rsp, err := s.process(r, nil, false)
	if err != nil {
		return err
	}
	w = rsp.(*raft.TimeoutNowResponse)
	return nil
}

func ServeHTTP(srv *Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-type", "application/json")
		service := rpc.NewServer()
		if err := service.Register(srv); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := service.ServeRequest(jsonrpc.NewServerCodec(&rpcServer{ReadCloser: r.Body, Writer: w})); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}

func Listen(srv *Service) error {
	parse, err := url.Parse(string(srv.addr))
	if err != nil {
		return err
	}
	//listen := &http.Server{
	//	Addr:    parse.Host,
	//	Handler: ServeHTTP(srv),
	//}
	//errCh := make(chan error)
	//go func() {
	//	errCh <- listen.ListenAndServe()
	//}()
	server := rpc.NewServer()
	server.Register(srv)
	listen, err := net.Listen("tcp", parse.Host)
	if err != nil {
		return err
	}
	defer listen.Close()
	go func() {
		server.Accept(listen)
	}()
	<-srv.ctx.Done()
	return context.Canceled
}

func NewService(ctx context.Context, id string, addr string) *Service {
	return &Service{
		id:          raft.ServerID(id),
		addr:        raft.ServerAddress(addr),
		ctx:         ctx,
		rpcCh:       make(chan raft.RPC),
		maxPipeline: 8,
	}
}
