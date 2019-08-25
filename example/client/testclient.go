package main

import (
	"context"
	"time"

	"github.com/huhuikevin/grpc-loadbalancer/balancer/picker"
	"github.com/huhuikevin/grpc-loadbalancer/example/proto"
	"github.com/huhuikevin/grpc-loadbalancer/resolver"
	"github.com/huhuikevin/grpc-loadbalancer/rpcclient"

	_ "github.com/huhuikevin/grpc-loadbalancer/balancer"

	"google.golang.org/grpc"
	_ "google.golang.org/grpc/balancer/roundrobin"
)

const (
	serverDomain = "example.test.com"
	resovler     = resolver.ResolverZookeeper
	blancePolicy = picker.RoundrobinBalanced
)

//TestClientWrapper wrapper for test client
type TestClientWrapper struct {
	rpcConn *grpc.ClientConn
	testClient proto.TestClient
	ctx context.Context
}

func NewClientTest(ctx context.Context) *TestClientWrapper {
	return &TestClientWrapper{
		ctx:ctx,
	}
}

//Start 开始
func (t *TestClientWrapper) Start() error {
	client, err := rpcclient.NewGRPCConnction(resovler, serverDomain, blancePolicy.String())
	if err != nil {
		return err
	}
	t.rpcConn = client
	t.testClient = proto.NewTestClient(t.rpcConn)
	return nil
}

//Say call Say function of testClient
func (t *TestClientWrapper) Say(message string, timeout time.Duration) (string, error) {
	ctx, _ := context.WithTimeout(t.ctx, timeout)
	resp, err := t.testClient.Say(ctx, &proto.SayReq{
		Content: message,
	})
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}


