package jsonrpc

import (
	"encoding/json"
	"github.com/hashicorp/raft"
)

type InstallSnapshotRequest struct {
	*raft.InstallSnapshotRequest
	Data json.RawMessage
}

//type AppendEntriesRequest raft.AppendEntriesRequest
//
//func (r *AppendEntriesRequest) MarshalJSON() ([]byte, error) {
//	buff, err := json.Marshal((*raft.AppendEntriesRequest)(r))
//	fmt.Println(string(buff), err)
//	return buff, err
//}
//
//func (r *AppendEntriesRequest) UnmarshalJSON(data []byte) error {
//	fmt.Println(string(data))
//	return json.Unmarshal(data, r)
//}
//
//type AppendEntriesResponse raft.AppendEntriesResponse
//
//func (r *AppendEntriesResponse) MarshalJSON() ([]byte, error) {
//	buff, err := json.Marshal((*raft.AppendEntriesResponse)(r))
//	fmt.Println(string(buff), err)
//	return buff, err
//}
//
//func (r *AppendEntriesResponse) UnmarshalJSON(data []byte) error {
//	fmt.Println(string(data))
//	return json.Unmarshal(data, r)
//}

//type rpcServer struct {
//	req io.ReadCloser
//	rsp http.ResponseWriter
//}
//
//func (r *rpcServer) Read(buff []byte) (int, error) {
//	var err error
//	buff, err = ioutil.ReadAll(r.req)
//	return len(buff), err
//}
//
//func (r *rpcServer) Write(buff []byte) (int, error) {
//	fmt.Println(string(buff))
//	//defer r.rsp.Header()
//	return r.rsp.Write(buff)
//}
//
//func (r *rpcServer) Close() error {
//	return r.req.Close()
//}
//
//type rpcClient struct {
//	ctx  context.Context
//	url  *url.URL
//	data chan io.ReadCloser
//}
//
//func (r *rpcClient) Read(buff []byte) (int, error) {
//	select {
//	case <-r.ctx.Done():
//		return 0, context.Canceled
//	case data := <-r.data:
//		defer data.Close()
//		var err error
//		buff, err = ioutil.ReadAll(data)
//		fmt.Printf("rsp:%s error:%s\n", string(buff), err)
//		if err != nil {
//			return 0, err
//		}
//		return len(buff), nil
//	}
//}
//
//func (r *rpcClient) Write(buff []byte) (int, error) {
//	fmt.Printf("Req(%s): %s\n", r.url.String(), string(buff))
//	req, err := http.NewRequestWithContext(r.ctx, "POST", r.url.String(), bytes.NewBuffer(buff))
//	if err != nil {
//		return 0, err
//	}
//	req.Header.Set("Content-type", "application/json")
//	rsp, err := http.DefaultClient.Do(req)
//	fmt.Printf("%+v\n", rsp.Header)
//	if err != nil {
//		return 0, err
//	}
//	if rsp.StatusCode != http.StatusOK {
//		return 0, errors.New(rsp.Status)
//	}
//	select {
//	case <-r.ctx.Done():
//		return 0, context.Canceled
//	case r.data <- rsp.Body:
//		return len(buff), nil
//	}
//}
//
//func (r *rpcClient) Close() error {
//	return nil
//}
