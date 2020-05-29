package jsonrpc

import (
	"io"
	"net/rpc"

	"github.com/hashicorp/raft"
)

type Transport struct {
	id    string
	addr  string
	rpcCh chan raft.RPC
}

func (t *Transport) getClient(id raft.ServerID, addr raft.ServerAddress) (*rpc.Client, error) {
	//raft.Transport()
	//raft.NewTCPTransport()
	return nil, nil
}

func (t *Transport) Consumer() <-chan raft.RPC {
	return t.rpcCh
}

func (t *Transport) LocalAddr() raft.ServerAddress {
	return raft.ServerAddress(t.addr)
}

func (t *Transport) AppendEntries(id raft.ServerID, addr raft.ServerAddress, args *raft.AppendEntriesRequest, resp *raft.AppendEntriesResponse) error {
	client, err := t.getClient(id, addr)
	if err != nil {
		return err
	}
	return client.Call("AppendEntries", args, resp)
}

func (t *Transport) AppendEntriesPipeline(id raft.ServerID, addr raft.ServerAddress) (raft.AppendPipeline, error) {
	return nil, nil
}

func (t *Transport) RequestVote(id raft.ServerID, addr raft.ServerAddress, args *raft.RequestVoteRequest, resp *raft.RequestVoteResponse) error {
	client, err := t.getClient(id, addr)
	if err != nil {
		return err
	}
	return client.Call("RequestVote", args, resp)
}

func (t *Transport) InstallSnapshot(id raft.ServerID, addr raft.ServerAddress, args *raft.InstallSnapshotRequest, resp *raft.InstallSnapshotResponse, data io.Reader) error {
	client, err := t.getClient(id, addr)
	if err != nil {
		return err
	}
	return client.Call("InstallSnapshot", args, resp)
}

func (t *Transport) TimeoutNow(id raft.ServerID, addr raft.ServerAddress, args *raft.TimeoutNowRequest, resp *raft.TimeoutNowResponse) error {
	client, err := t.getClient(id, addr)
	if err != nil {
		return err
	}
	return client.Call("TimeoutNow", args, resp)
}

func (t *Transport) EncodePeer(id raft.ServerID, addr raft.ServerAddress) []byte {
	return []byte(addr)
}

func (t *Transport) DecodePeer(addr []byte) raft.ServerAddress {
	return raft.ServerAddress(addr)
}

func (t *Transport) SetHeartbeatHandler(cb func(rpc raft.RPC)) {

}
