package main

import (
	"flag"
	"fmt"

	"github.com/huhuikevin/grpc-loadbalancer/resolver"
	_ "github.com/huhuikevin/grpc-loadbalancer/resolver/etcd"
)

var port = flag.Int("port", 4000, "listening port")

var weight = flag.Int("weight", 1, "weight")

func init() {
	resolver.AddNameServers(resovlerName, []string{"http://localhost:2379"})
}

func startService() {
	address := fmt.Sprintf("0.0.0.0:%d", *port)
	extaddr := fmt.Sprintf("localhost:%d", *port)
	StartServer(address, extaddr, int32(*weight), 0)
}

//go run main.go -weight 50 -port 28544
//go run main.go -weight 1 -port 18562
//go run main.go -weight 100 -port 27772
func main() {
	flag.Parse()
	startService()
}
