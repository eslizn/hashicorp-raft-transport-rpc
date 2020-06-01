package jsonrpc

import (
	"context"
	"fmt"
	"net"
	"net/rpc"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/raft"
)

type Rpc struct{}

func (r *Rpc) AppendEntries(req raft.AppendEntriesRequest, rsp *raft.AppendEntriesResponse) error {
	rsp.Success = true
	rsp.RPCHeader = req.RPCHeader
	return nil
}

func TestRPC(t *testing.T) {
	srv := rpc.NewServer()
	err := srv.Register(&Rpc{})
	if err != nil {
		t.Error(err)
		return
	}
	t.Log("init")
	//jsonrpc.
	//go http.ListenAndServe("127.0.0.1:8080", srv)
	go func() {
		listen, err := net.Listen("tcp", ":8080")
		if err != nil {
			panic(err)
		}
		srv.Accept(listen)
		//for {
		//	conn, err := listen.Accept()
		//	if err != nil {
		//		//t.Error(err)
		//		continue
		//	}
		//	go srv.ServeConn(conn)
		//}
	}()
	time.Sleep(time.Second)
	t.Log("start dial")
	client, err := rpc.Dial("tcp", "127.0.0.1:8080")
	t.Log("done dial")
	if err != nil {
		t.Error(err)
		return
	}
	rsp := raft.AppendEntriesResponse{}
	err = client.Call("Rpc.AppendEntries", raft.AppendEntriesRequest{}, &rsp)
	if err != nil {
		t.Error(err)
		return
	}
	t.Logf("%+v\n", rsp)
}

type Rafts []*raft.Raft

func (rs Rafts) Leader(ctx context.Context) *raft.Raft {
	for {
		select {
		case <-time.NewTimer(time.Second).C:
			for k := range rs {
				if rs[k] != nil && rs[k].State() == raft.Leader {
					return rs[k]
				}
			}
		case <-ctx.Done():
			return nil
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
		rafts       = make(Rafts, len(addrs))
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
			go srv.Accept(listen)
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
			leader := rafts.Leader(ctx)
			if leader == nil {
				t.Error("not found leader")
				return
			}
			//t.Log(rafts[0].State())
			future := leader.AddVoter(raft.ServerID(fmt.Sprint(k)), raft.ServerAddress(addrs[k]), 0, time.Minute)
			if future.Error() != nil {
				//t.Errorf("AddVoter(%d, %s) error: %s\n", k, addrs[k], future.Error())
				//return
			}
		}
	}
	//fmt.Println("start check")
	t.Logf("leader: %+v\n", rafts.Leader(ctx))
	for k := range rafts {
		t.Logf("%d %s ---> leader: %s\n", k, rafts[k].State(), rafts[k].Leader())
		t.Logf("%s %+v\n", rafts[k].State(), rafts[k].Apply([]byte("test"), time.Second).Error())
	}
	//time.Sleep(5 * time.Second)
	//t.Logf("%+v\n", rafts.Leader(ctx).GetConfiguration().Configuration().Servers)
	time.Sleep(time.Minute)
}

func TestTCPTransport(t *testing.T) {
	var (
		addrs = []string{
			"tcp://127.0.0.1:10001/",
			"tcp://127.0.0.1:10002/",
			"tcp://127.0.0.1:10003/",
		}
		rafts       = make([]*raft.Raft, len(addrs))
		ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	)
	defer cancel()
	for k := range addrs {
		var (
			parse, _   = url.Parse(addrs[k])
			addr, _    = net.ResolveTCPAddr(parse.Scheme, parse.Host)
			config     = raft.DefaultConfig()
			fsm        = &raft.MockFSM{}
			logs       = raft.NewInmemStore()
			stable     = raft.NewInmemStore()
			snapshot   = raft.NewInmemSnapshotStore()
			trans, err = raft.NewTCPTransport(parse.Host, addr, 8, 0, os.Stdout)
		)
		if err != nil {
			t.Error(err)
			return
		}
		config.LogLevel = "WARN" //"WARN"
		config.LocalID = raft.ServerID(fmt.Sprint(k))
		rafts[k], err = raft.NewRaft(config, fsm, logs, stable, snapshot, trans)
		if err != nil {
			t.Log(err)
			return
		}
		if k == 0 {
			rafts[k].BootstrapCluster(raft.Configuration{
				Servers: []raft.Server{
					{
						ID:      config.LocalID,
						Address: raft.ServerAddress(parse.Host),
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
			future := rafts[0].AddVoter(raft.ServerID(fmt.Sprint(k)), raft.ServerAddress(parse.Host), 0, 0)
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
