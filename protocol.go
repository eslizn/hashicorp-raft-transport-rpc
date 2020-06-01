package rpctrans

import (
	"encoding/json"
	"github.com/hashicorp/raft"
)

type InstallSnapshotRequest struct {
	*raft.InstallSnapshotRequest
	Data json.RawMessage
}
