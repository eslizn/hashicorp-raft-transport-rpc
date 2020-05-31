package jsonrpc

import "github.com/hashicorp/raft"

type InstallSnapshotRequest struct {
	*raft.InstallSnapshotRequest
	Data []byte
}
