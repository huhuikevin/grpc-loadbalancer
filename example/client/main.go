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
	resolver.AddNameServers(resovler, []string{"192.168.3.45:32350"})
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
		go func() {
			defer wg.Done()
			resp, err1 := test.Say("round robin", 5*time.Second)
			if err1 != nil {
				logtest.Error(logs.Error(err1))
				time.Sleep(time.Second)
				return
			}
			logtest.Info(logs.String("Recved", resp))
		}()
	}
	wg.Wait()
	//test.Print()
}

// func main() {
// 	c, err := grpc.Dial(resovler+":///"+serverDomain, grpc.WithInsecure(), grpc.WithBalancerName("wroundrobin-balanced"), grpc.WithTimeout(time.Second*5))
// 	if err != nil {
// 		log.Printf("grpc dial: %s", err)
// 		return
// 	}
// 	defer c.Close()
// 	log.Println("start test........")
// 	client := proto.NewTestClient(c)
// 	for i := 0; i < 5000; i++ {
// 		ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
// 		resp, err := client.Say(ctx, &proto.SayReq{Content: "round robin"})
// 		if err != nil {
// 			log.Println("error:", err)
// 			time.Sleep(time.Second)
// 			continue
// 		}
// 		time.Sleep(time.Second)
// 		//time.Sleep(time.Second * 10000)
// 		log.Printf(resp.Content)
// 	}

// }
