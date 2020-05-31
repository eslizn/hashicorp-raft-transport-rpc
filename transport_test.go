package jsonrpc

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/raft"
)

func TestTransport(t *testing.T) {
	var (
		addrs = []string{
			"http://127.0.0.1:10001/raft/",
			"http://127.0.0.1:10002/raft/",
			"http://127.0.0.1:10003/raft/",
		}
		rafts       = make([]*raft.Raft, len(addrs))
		ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	)
	defer cancel()
	for k := range addrs {
		var (
			err      error
			config   = raft.DefaultConfig()
			fsm      = &raft.MockFSM{}
			logs     = raft.NewInmemStore()
			stable   = raft.NewInmemStore()
			snapshot = raft.NewInmemSnapshotStore()
			trans    = NewService(ctx, fmt.Sprint(k), addrs[k])
		)
		go func() {
			err := Listen(trans)
			if err != nil {
				t.Error(err)
			}
		}()
		config.LogLevel = "ERROR" //"WARN"
		config.LocalID = raft.ServerID(fmt.Sprint(k))
		rafts[k], err = raft.NewRaft(config, fsm, logs, stable, snapshot, (*Transport)(trans))
		if err != nil {
			t.Log(err)
			return
		}
		if k == 0 {
			rafts[k].BootstrapCluster(raft.Configuration{
				Servers: []raft.Server{
					{
						ID:      config.LocalID,
						Address: raft.ServerAddress(addrs[k]),
					},
				},
			})
			for rafts[k].State() != raft.Leader {
				select {
				case <-ctx.Done():
					return
				case <-time.NewTimer(time.Second).C:
					continue
				}
			}
		} else {
			future := rafts[0].AddVoter(raft.ServerID(fmt.Sprint(k)), raft.ServerAddress(addrs[k]), 0, 0)
			if future.Error() != nil {
				t.Logf("AddVoter(%d, %s) error: %s\n", k, addrs[k], future.Error())
			}

		}
	}
	//fmt.Println("start check")
	//ready := false
	//for !ready {
	//	select {
	//	case <-time.NewTimer(time.Second).C:
	//		for k := range rafts {
	//			if rafts[k].State() == raft.Leader {
	//				ready = true
	//				break
	//			}
	//		}
	//	case <-ctx.Done():
	//		return
	//	}
	//}
	for k := range rafts {
		t.Logf("%d %s ---> leader: %s\n", k, rafts[k].State(), rafts[k].Leader())
		t.Logf("%s %+v\n", rafts[k].State(), rafts[k].Apply([]byte("test"), time.Second).Error())
	}
	time.Sleep(5 * time.Second)
	t.Logf("%+v\n", rafts[0].GetConfiguration().Configuration().Servers)

	//time.Sleep(time.Minute)
}
