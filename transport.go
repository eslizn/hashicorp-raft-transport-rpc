package rpctrans

import (
	"context"
	"github.com/hashicorp/raft"
	"io"
	"io/ioutil"
	"net/rpc"
	"net/url"
	"strings"
	"sync"
)

type Transport struct {
	id              raft.ServerID
	addr            raft.ServerAddress
	ctx             context.Context
	rpcCh           chan raft.RPC
	heartbeatFn     func(raft.RPC)
	heartbeatFnLock sync.RWMutex
	maxPipeline     int
	clients         sync.Map
}

func (t *Transport) GetClient(id raft.ServerID, addr raft.ServerAddress) (*rpc.Client, error) {
	//@todo check already join cluster
	client, load := t.clients.Load(addr)
	if !load {
		parse, err := url.Parse(string(addr))
		if err != nil {
			return nil, err
		}
		client, err = rpc.DialHTTPPath("tcp", parse.Host, parse.Path)
		if err != nil {
			return nil, err
		}
		t.clients.Store(addr, client)
	}
	return client.(*rpc.Client), nil
}

func (t *Transport) Consumer() <-chan raft.RPC {
	return t.rpcCh
}

func (t *Transport) LocalAddr() raft.ServerAddress {
	return t.addr
}

func (t *Transport) AppendEntries(id raft.ServerID, addr raft.ServerAddress, args *raft.AppendEntriesRequest, resp *raft.AppendEntriesResponse) error {
	client, err := t.GetClient(id, addr)
	if err != nil {
		return err
	}
	return client.Call("Service.AppendEntries", args, resp)
}

func (t *Transport) AppendEntriesPipeline(id raft.ServerID, addr raft.ServerAddress) (raft.AppendPipeline, error) {
	client, err := t.GetClient(id, addr)
	if err != nil {
		return nil, err
	}
	p := &pipeline{
		client:      client,
		doneCh:      make(chan raft.AppendFuture, t.maxPipeline),
		progressCh:  make(chan *future, t.maxPipeline),
		maxPipeline: t.maxPipeline,
	}
	p.ctx, p.cancelCtx = context.WithCancel(t.ctx)
	go func() {
		err = p.progress()
	}()
	return p, err
}

func (t *Transport) RequestVote(id raft.ServerID, addr raft.ServerAddress, args *raft.RequestVoteRequest, resp *raft.RequestVoteResponse) error {
	client, err := t.GetClient(id, addr)
	if err != nil {
		return err
	}
	return client.Call("Service.RequestVote", args, resp)
}

func (t *Transport) InstallSnapshot(id raft.ServerID, addr raft.ServerAddress, args *raft.InstallSnapshotRequest, resp *raft.InstallSnapshotResponse, data io.Reader) error {
	client, err := t.GetClient(id, addr)
	if err != nil {
		return err
	}
	req := &InstallSnapshotRequest{
		InstallSnapshotRequest: args,
	}
	req.Data, err = ioutil.ReadAll(data)
	if err != nil {
		return err
	}
	return client.Call("Service.InstallSnapshot", req, resp)
}

func (t *Transport) TimeoutNow(id raft.ServerID, addr raft.ServerAddress, args *raft.TimeoutNowRequest, resp *raft.TimeoutNowResponse) error {
	client, err := t.GetClient(id, addr)
	if err != nil {
		return err
	}
	return client.Call("Service.TimeoutNow", args, resp)
}

func (t *Transport) EncodePeer(id raft.ServerID, addr raft.ServerAddress) []byte {
	return []byte(strings.Join([]string{string(id), string(addr)}, "|"))
}

func (t *Transport) DecodePeer(data []byte) raft.ServerAddress {
	list := strings.Split(string(data), "|")
	if len(list) > 1 {
		return raft.ServerAddress(strings.Join(list[1:], "|"))
	}
	return ""
}

func (t *Transport) SetHeartbeatHandler(cb func(rpc raft.RPC)) {
	t.heartbeatFnLock.Lock()
	defer t.heartbeatFnLock.Unlock()
	t.heartbeatFn = cb
}
