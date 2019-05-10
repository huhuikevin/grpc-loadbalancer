package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/huhuikevin/grpc-loadbalancer/balancer/picker"
	"github.com/huhuikevin/grpc-loadbalancer/example/proto"
	"github.com/huhuikevin/grpc-loadbalancer/logs"
	"github.com/huhuikevin/grpc-loadbalancer/resolver"
	"github.com/huhuikevin/grpc-loadbalancer/rpcclient"

	_ "github.com/huhuikevin/grpc-loadbalancer/balancer"

	"google.golang.org/grpc"
)

const (
	serverDomain = "example.test.com"
	resovler     = resolver.ResolverETCD3
	blancePolicy = picker.WeightedRoundrobinBalanced
)

//TestClientWrapper wrapper for test client
type TestClientWrapper struct {
	caller *rpcclient.Caller
	calls  map[string]int64
	mu     sync.Mutex
}

//NewClient 创建一个真实的rpc client，必须要实现GRPCClient 的NewClient接口
//用户不需要调用
func (t *TestClientWrapper) NewClient(conn *grpc.ClientConn) interface{} {
	return proto.NewTestClient(conn)
}

//Start 开始
func (t *TestClientWrapper) Start() error {
	caller := rpcclient.NewCaller(resovler, serverDomain, blancePolicy.String(), 1000)
	err := caller.Start(t)
	if err != nil {
		return err
	}
	t.caller = caller
	t.calls = make(map[string]int64)
	t.mu = sync.Mutex{}
	return nil
}

//Say call Say function of testClient
func (t *TestClientWrapper) Say(in *proto.SayReq, timeout time.Duration) (*proto.SayResp, error) {
	in.Content = blancePolicy.String()
	v, err := t.caller.InvokeWithArgs2("Say", []interface{}{in, []grpc.CallOption{}}, timeout)
	if err != nil {
		logtest.Error(logs.Error(err))
		return nil, err
	}

	resp, ok := v.(*proto.SayResp)
	if !ok {
		fmt.Printf("resp : %v\r\n", resp)
		return nil, rpcclient.ErrReturnValueCanNotConvertToStruct
	}
	t.mu.Lock()
	t.calls[resp.Content] = t.calls[resp.Content] + 1
	t.mu.Unlock()
	return resp, nil
}

//Print debug
func (t *TestClientWrapper) Print() {
	for k, v := range t.calls {
		logtest.Info(k, logs.Int64("calls", v))
	}
}
