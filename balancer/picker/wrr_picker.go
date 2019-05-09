// Copyright 2018 The etcd Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package picker

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/huhuikevin/grpc-loadbalancer/logs"
	myresolver "github.com/huhuikevin/grpc-loadbalancer/resolver"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/resolver"
)

const (
	defaultWeight = 100
)

var (
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
)

type addressItem struct {
	weight      int32
	disable     int32
	calls       int64
	weightRatio float32
	callRatio   float32
	subcon      balancer.SubConn
	address     resolver.Address
}

func (ai addressItem) string() string {
	return fmt.Sprintf("address:%s weight:%d, subcon %p", ai.address.Addr, ai.weight, ai.subcon)
}

// NewWRRBalanced returns a new roundrobin balanced picker.
func NewWRRBalanced(addrToSc map[resolver.Address]balancer.SubConn) Picker {
	addrs := make([]addressItem, 0, len(addrToSc))
	weighted0 := make([]addressItem, 0, len(addrToSc))
	sumOfWeight := int64(0)
	for addres, subcon := range addrToSc {
		addrItem := addressItem{
			weight:  defaultWeight,
			subcon:  subcon,
			address: addres,
		}
		if addres.Metadata != nil {
			minfo, ok := addres.Metadata.(myresolver.BalanceData)
			if ok {
				addrItem.weight = minfo.Weight
				addrItem.disable = minfo.Disable
			}
		}
		sumOfWeight += int64(addrItem.weight)
		if addrItem.disable != 0 {
			weighted0 = append(weighted0, addrItem)
		} else {
			addrs = append(addrs, addrItem)
		}
		log.Info(addrItem.string())
	}
	for i := 0; i < len(addrs); i++ {
		item := &addrs[i]
		item.weightRatio = float32(item.weight) / float32(sumOfWeight)
	}
	return &wrrBalanced{
		weighted0:    weighted0,
		addrs:        addrs,
		sumOfWeights: sumOfWeight,
	}
}

type wrrBalanced struct {
	mu sync.RWMutex
	//next  int
	//scs   []balancer.SubConn
	addrs []addressItem
	//存储weight 为0的地址，这些地址，不参与服务
	weighted0 []addressItem
	//addrToSc map[resolver.Address]balancer.SubConn
	//scToAddr     map[balancer.SubConn]resolver.Address
	sumOfWeights int64
	sumOfCalls   int64
}

func (rb *wrrBalanced) getSubCon() *addressItem {
	max := int32(len(rb.addrs))
	cur := r.Int31n(max)
	if rb.sumOfCalls == 0 {
		return &rb.addrs[cur]
	}
	for idx, item := range rb.addrs {
		if idx == int(cur) {
			continue
		}
		callRatio := float32(item.calls) / float32(rb.sumOfCalls)
		if callRatio < item.weightRatio {
			cur = int32(idx)
			break
		}
	}
	return &rb.addrs[cur]
}

func (rb *wrrBalanced) printInfo() {
	log.Info(logs.Int64("sumOfcalls", rb.sumOfCalls))
	for _, item := range rb.addrs {
		log.Info("addr=", item.address.Addr, logs.Int64("calls", item.calls))
	}
}

// Pick is called for every client request.
func (rb *wrrBalanced) Pick(ctx context.Context, opts balancer.PickOptions) (balancer.SubConn, func(balancer.DoneInfo), error) {
	rb.mu.RLock()
	n := len(rb.addrs)
	rb.mu.RUnlock()
	if n == 0 {
		return nil, nil, balancer.ErrNoSubConnAvailable
	}

	rb.mu.Lock()
	item := rb.getSubCon()
	item.calls++
	rb.sumOfCalls++
	sc := item.subcon
	rb.mu.Unlock()
	//log.Info("picker:", picked)
	//rb.printInfo()
	doneFunc := func(info balancer.DoneInfo) {
		// TODO: error handling?
		// log.Info("wrr balancer",
		// 	logs.Bool("success", info.Err == nil),
		// 	logs.Error(info.Err),
		// 	logs.String("address", item.address.Addr),
		// 	logs.Bool("bytes-sent", info.BytesSent),
		// 	logs.Bool("bytes-recv", info.BytesReceived))
	}
	return sc, doneFunc, nil
}
