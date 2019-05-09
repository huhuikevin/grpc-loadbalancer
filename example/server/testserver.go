package main

import (
	"log"

	"github.com/huhuikevin/grpc-loadbalancer/example/proto"
	"github.com/huhuikevin/grpc-loadbalancer/resolver"
	"github.com/huhuikevin/grpc-loadbalancer/rpcserver"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type testserver struct {
	server *rpcserver.Server
	nodeID string
}

const (
	serverDomain = "example.test.com"
	resovlerName = resolver.ResolverETCD3
)

//StartServer start the test server
func StartServer(address string, extAddress string, weight int32, disable int32) error {
	server := rpcserver.New(rpcserver.Config{
		Resolver:   resovlerName,
		Domain:     serverDomain,
		Address:    address,
		ExtAddress: extAddress,
		Weight:     weight,
		Disable:    disable,
	})
	return server.Start(&testserver{server: server})
}

func (s *testserver) RegisterToGRPC(grpcs *grpc.Server) {
	proto.RegisterTestServer(grpcs, s)
}

//Say say helle
func (s *testserver) Say(ctx context.Context, req *proto.SayReq) (*proto.SayResp, error) {
	text := "Hello " + req.Content + ", I am " + s.server.NodeID()
	log.Println(text)
	//time.Sleep(6 * time.Second)
	return &proto.SayResp{Content: text}, nil
}
