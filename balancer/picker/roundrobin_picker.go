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
	"sync"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/resolver"
)

// NewRoundrobinBalanced returns a new roundrobin balanced picker.
func NewRoundrobinBalanced(addrToSc map[resolver.Address]balancer.SubConn) Picker {
	scs := make([]balancer.SubConn, 0, len(addrToSc))
	newScToAddr := make(map[balancer.SubConn]resolver.Address)
	for addr, sc := range addrToSc {
		scs = append(scs, sc)
		newScToAddr[sc] = addr
	}

	return &rrBalanced{
		scs:      scs,
		scToAddr: newScToAddr,
	}
}

type rrBalanced struct {
	mu   sync.RWMutex
	next int
	scs  []balancer.SubConn

	//addrToSc map[resolver.Address]balancer.SubConn
	scToAddr map[balancer.SubConn]resolver.Address
}

// Pick is called for every client request.
func (rb *rrBalanced) Pick(ctx context.Context, opts balancer.PickOptions) (balancer.SubConn, func(balancer.DoneInfo), error) {
	rb.mu.RLock()
	n := len(rb.scs)
	rb.mu.RUnlock()
	if n == 0 {
		return nil, nil, balancer.ErrNoSubConnAvailable
	}

	rb.mu.Lock()
	cur := rb.next
	sc := rb.scs[cur]
	rb.next = (rb.next + 1) % len(rb.scs)
	rb.mu.Unlock()

	doneFunc := func(info balancer.DoneInfo) {
		// TODO: error handling?
		// log.Info("rr balancer",
		// 	logs.Bool("success", info.Err == nil),
		// 	logs.Error(info.Err),
		// 	logs.Bool("bytes-sent", info.BytesSent),
		// 	logs.Bool("bytes-recv", info.BytesReceived))
	}
	return sc, doneFunc, nil
}
