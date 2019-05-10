package rpcclient

import "google.golang.org/grpc"

//GRPCClient the rpc client must implement this interface
//在创建caller的时候回自动调用，返回一个真实的rpc client
type GRPCClient interface {
	NewClient(*grpc.ClientConn) interface{}
}
