package rpcclient

import (
	"google.golang.org/grpc"
	"log"
)

func NewGRPCConnction(resolver string, domain string, balance string) (*grpc.ClientConn, error) {
	//target is for the naming finder,example etcd:///test.example.com
	//the grpc will use the naming server of "etcd" for name resolver
	var target string = ""
	if resolver != "" {
		target = resolver + ":///" + domain
	}else {
		target = domain
	}
	log.Print("target=", target)
	dialOpts := make ([]grpc.DialOption, 0, 0)
	dialOpts = append(dialOpts, grpc.WithInsecure())
	if balance != "" {
		dialOpts = append(dialOpts, grpc.WithBalancerName(balance))
	}
	client, err := grpc.Dial(target, dialOpts...)
	if err != nil {
		return nil, err
	}
	return client, nil
}