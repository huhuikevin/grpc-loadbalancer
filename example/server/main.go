package main

import (
	"flag"
	"fmt"
	"github.com/huhuikevin/grpc-loadbalancer/utils"

	"github.com/huhuikevin/grpc-loadbalancer/resolver"
	_ "github.com/huhuikevin/grpc-loadbalancer/resolver/zookeeper"
)

//var port = flag.Int("port", 8080, "listening port")
//
//var weight = flag.Int("weight", 1, "weight")

func init() {
	resolver.AddNameServers(resovlerName, []string{"kafka-zookeeper-headless.kafka-cluster:2181"})
}

func startService() {
	port := 8080
	weight := 100
	ip, err := utils.GetLocalIp()
	if err != nil {
		fmt.Println("get localip", err)
	}
	address := fmt.Sprintf("0.0.0.0:%d", port)
	extaddr := fmt.Sprintf("%s:%d", ip, port)
	StartServer(address, extaddr, int32(weight), 0)
}

//go run main.go -weight 50 -port 28544
//go run main.go -weight 1 -port 18562
//go run main.go -weight 100 -port 27772
func main() {
	flag.Parse()
	startService()
}
