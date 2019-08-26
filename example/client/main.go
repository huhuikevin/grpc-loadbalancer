package main

import (
	"context"
	"sync"
	"time"

	"github.com/huhuikevin/grpc-loadbalancer/logs"
	"github.com/huhuikevin/grpc-loadbalancer/resolver"
	_ "github.com/huhuikevin/grpc-loadbalancer/resolver/zookeeper"
)

var logtest = &logs.SimpleLog{Level: logs.DebugLvl}

func init() {
	resolver.AddNameServers(resovler, []string{"kafka-zookeeper-headless.kafka-cluster:2181"})
}

func main() {
	test := NewClientTest(context.Background())
	err := test.Start()
	if err != nil {
		logtest.Error(logs.Error(err))
	}
	time.Sleep(time.Second * 1)
	wg := sync.WaitGroup{}
	for i := 0; i < 10000; i++ {
		wg.Add(1)

			resp, err1 := test.Say("round robin", 5*time.Second)
			if err1 != nil {
				logtest.Error(logs.Error(err1))
				time.Sleep(time.Second)
				return
			}
			logtest.Info(logs.String("Recved", resp))
			time.Sleep(time.Second)

	}
	wg.Wait()
	//test.Print()
}

