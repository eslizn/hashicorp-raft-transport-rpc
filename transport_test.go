package rpctrans

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/rpc"
	"net/url"
	"testing"
	"time"

	"github.com/hashicorp/raft"
)

func getLeader(ctx context.Context, rafts []*raft.Raft) (*raft.Raft, error) {
	for {
		select {
		case <-time.NewTimer(time.Second).C:
			for k := range rafts {
				if rafts[k] != nil && rafts[k].State() == raft.Leader {
					return rafts[k], nil
				}
			}
		case <-ctx.Done():
			return nil, context.Canceled
		}
	}
}

func TestTransport(t *testing.T) {
	var (
		addrs = []string{
			"http://127.0.0.1:10001/",
			"http://127.0.0.1:10002/",
			"http://127.0.0.1:10003/",
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
			parse, err := url.Parse(string(trans.addr))
			if err != nil {
				t.Error(err)
				return
			}
			srv := rpc.NewServer()
			err = srv.Register(trans)
			if err != nil {
				t.Error(err)
				return
			}
			listen, err := net.Listen("tcp", parse.Host)
			if err != nil {
				t.Error(err)
				return
			}
			go http.Serve(listen, srv)
			<-trans.ctx.Done()
		}()
		time.Sleep(time.Second)
		config.LeaderLeaseTimeout = 3 * time.Second
		config.CommitTimeout = 3 * time.Second
		config.ElectionTimeout = 3 * time.Second
		config.HeartbeatTimeout = 3 * time.Second
		config.LogLevel = "WARN"
		config.LocalID = raft.ServerID(fmt.Sprint(k))
		rafts[k], err = raft.NewRaft(config, fsm, logs, stable, snapshot, (*Transport)(trans))
		if err != nil {
			t.Error(err)
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
		} else {
			leader, err := getLeader(ctx, rafts[0:k])
			if err != nil {
				t.Error(err)
				return
			}
			f := leader.AddVoter(raft.ServerID(fmt.Sprint(k)), raft.ServerAddress(addrs[k]), 0, time.Minute)
			if f.Error() != nil {
				t.Errorf("AddVoter(%d, %s) error: %s\n", k, addrs[k], f.Error())
				return
			}
		}
	}
	for k := range rafts {
		t.Logf("%d %s ---> leader: %s\n", k, rafts[k].State(), rafts[k].Leader())
		t.Logf("%s %+v\n", rafts[k].State(), rafts[k].Apply([]byte("test"), time.Second).Error())
	}
}
