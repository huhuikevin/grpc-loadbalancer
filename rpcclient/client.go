package rpcclient

import (
	"google.golang.org/grpc"
	"log"
)

func NewGRPCConnction(resolver string, domain string, balance string) (*grpc.ClientConn, error) {
	//target is for the naming finder,example etcd:///test.example.com
	//the grpc will use the naming server of "etcd" for name resolver
	target := resolver + ":///" + domain
	log.Print("target=", target)
	client, err := grpc.Dial(target, grpc.WithInsecure(), grpc.WithBalancerName(balance))
	if err != nil {
		return nil, err
	}
	return client, nil
}