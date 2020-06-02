package rpctrans

import (
	"encoding/json"
	"github.com/hashicorp/raft"
)

//merge data to InstallSnapshotRequest
type InstallSnapshotRequest struct {
	*raft.InstallSnapshotRequest
	Data json.RawMessage
}
