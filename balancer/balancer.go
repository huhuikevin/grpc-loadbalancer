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

package balancer

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/huhuikevin/grpc-loadbalancer/logs"

	"github.com/huhuikevin/grpc-loadbalancer/balancer/picker"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/resolver"
	_ "google.golang.org/grpc/resolver/dns"         // register DNS resolver
	_ "google.golang.org/grpc/resolver/passthrough" // register passthrough resolver
)

var log = logs.SLog

// RegisterBuilder creates and registers a builder. Since this function calls balancer.Register, it
// must be invoked at initialization time.
func RegisterBuilder(policy picker.Policy) {
	bb := &balanceBuilder{policy}
	balancer.Register(bb)
}

type balanceBuilder struct {
	policy picker.Policy
}

// Build is called initially when creating "ccBalancerWrapper".
// "grpc.Dial" is called to this client connection.
// Then, resolved addresses will be handled via "HandleResolvedAddrs".
func (b *balanceBuilder) Build(cc balancer.ClientConn, opt balancer.BuildOptions) balancer.Balancer {
	bb := &baseBalancer{
		policy:   b.policy,
		name:     b.policy.String(),
		addrToSc: make(map[resolver.Address]balancer.SubConn),
		scToAddr: make(map[balancer.SubConn]resolver.Address),
		scToSt:   make(map[balancer.SubConn]connectivity.State),

		currentConn: nil,
		csEvltr:     &connectivityStateEvaluator{},

		// initialize picker always returns "ErrNoSubConnAvailable"
		Picker: picker.NewErr(balancer.ErrNoSubConnAvailable),
	}

	// TODO: support multiple connections
	bb.mu.Lock()
	bb.currentConn = cc
	bb.mu.Unlock()

	return bb
}

// Name implements "grpc/balancer.Builder" interface.
func (b *balanceBuilder) Name() string { return b.policy.String() }

// Balancer defines client balancer interface.
type Balancer interface {
	// Balancer is called on specified client connection. Client initiates gRPC
	// connection with "grpc.Dial(addr, grpc.WithBalancerName)", and then those resolved
	// addresses are passed to "grpc/balancer.Balancer.HandleResolvedAddrs".
	// For each resolved address, balancer calls "balancer.ClientConn.NewSubConn".
	// "grpc/balancer.Balancer.HandleSubConnStateChange" is called when connectivity state
	// changes, thus requires failover logic in this method.
	balancer.Balancer

	// Picker calls "Pick" for every client request.
	balancer.Picker
}

type baseBalancer struct {
	name   string
	policy picker.Policy
	mu     sync.RWMutex

	addrToSc map[resolver.Address]balancer.SubConn
	scToAddr map[balancer.SubConn]resolver.Address
	scToSt   map[balancer.SubConn]connectivity.State

	currentConn  balancer.ClientConn
	currentState connectivity.State
	csEvltr      *connectivityStateEvaluator

	balancer.Picker
}

func addrsToStrings(addrs []resolver.Address) string {
	address := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		address = append(address, addr.Addr)
	}
	return strings.Join(address, ",")
}
func scToString(sc balancer.SubConn) string {
	return fmt.Sprintf("%p", sc)
}

func scsToStrings(scs map[balancer.SubConn]resolver.Address) string {
	ss := make([]string, 0, len(scs))
	for sc, a := range scs {
		ss = append(ss, fmt.Sprintf("%s (%s)", a.Addr, scToString(sc)))
	}
	sort.Strings(ss)

	return strings.Join(ss, ",")
}

// HandleResolvedAddrs implements "grpc/balancer.Balancer" interface.
// gRPC sends initial or updated resolved addresses from "Build".
func (bb *baseBalancer) HandleResolvedAddrs(addrs []resolver.Address, err error) {
	if err != nil {
		log.Warn("HandleResolvedAddrs called with error", logs.String("balancer-name", bb.name), logs.Error(err))
		//bb.lg.Warn("HandleResolvedAddrs called with error", zap.String("balancer-id", bb.id), zap.Error(err))
		return
	}
	log.Info("resolved", logs.String("balancer-name", bb.name), logs.String("addresses", addrsToStrings(addrs)))

	bb.mu.Lock()
	defer bb.mu.Unlock()
	resolved := make(map[resolver.Address]struct{})
	for _, addr := range addrs {
		resolved[addr] = struct{}{}
		if _, ok := bb.addrToSc[addr]; !ok {
			sc, err := bb.currentConn.NewSubConn([]resolver.Address{addr}, balancer.NewSubConnOptions{})
			if err != nil {
				log.Warn("NewSubConn failed", logs.String("balancer-name", bb.name), logs.Error(err), logs.String("address", addr.Addr))
				continue
			}
			bb.addrToSc[addr] = sc
			bb.scToAddr[sc] = addr
			bb.scToSt[sc] = connectivity.Idle
			sc.Connect()
		}
	}

	for addr, sc := range bb.addrToSc {
		if _, ok := resolved[addr]; !ok {
			// was removed by resolver or failed to create subconn
			bb.currentConn.RemoveSubConn(sc)
			delete(bb.addrToSc, addr)

			log.Info(
				"removed subconn",
				logs.String("balancer-name", bb.name),
				logs.String("address", addr.Addr),
				logs.String("subconn", scToString(sc)),
			)

			// Keep the state of this sc in bb.scToSt until sc's state becomes Shutdown.
			// The entry will be deleted in HandleSubConnStateChange.
			// (DO NOT) delete(bb.scToAddr, sc)
			// (DO NOT) delete(bb.scToSt, sc)
		}
	}
}

// HandleSubConnStateChange implements "grpc/balancer.Balancer" interface.
func (bb *baseBalancer) HandleSubConnStateChange(sc balancer.SubConn, s connectivity.State) {
	bb.mu.Lock()
	defer bb.mu.Unlock()

	old, ok := bb.scToSt[sc]
	if !ok {
		log.Warn(
			"state change for an unknown subconn",
			logs.String("balancer-name", bb.name),
			logs.String("subconn", scToString(sc)),
			logs.String("state", s.String()),
		)
		return
	}

	log.Info(
		"subcon state changed",
		logs.Bool("connected", s == connectivity.Ready),
		logs.String("address", bb.scToAddr[sc].Addr),
		logs.String("old-state", old.String()),
		logs.String("new-state", s.String()),
	)

	bb.scToSt[sc] = s
	switch s {
	case connectivity.Idle:
		sc.Connect()
	case connectivity.Shutdown:
		// When an address was removed by resolver, b called RemoveSubConn but
		// kept the sc's state in scToSt. Remove state for this sc here.
		delete(bb.scToAddr, sc)
		delete(bb.scToSt, sc)
	}

	oldAggrState := bb.currentState
	bb.currentState = bb.csEvltr.recordTransition(old, s)
	log.Info(
		"balance state changed",
		logs.String("balancer-name", bb.name),
		logs.Bool("connected", bb.currentState == connectivity.Ready),
		logs.Int32("subconn-size", int32(len(bb.scToAddr))),
		logs.String("address", bb.scToAddr[sc].Addr),
		logs.String("old-state", oldAggrState.String()),
		logs.String("new-state", bb.currentState.String()),
	)
	// Regenerate picker when one of the following happens:
	//  - this sc became ready from not-ready
	//  - this sc became not-ready from ready
	//  - the aggregated state of balancer became TransientFailure from non-TransientFailure
	//  - the aggregated state of balancer became non-TransientFailure from TransientFailure
	if (s == connectivity.Ready) != (old == connectivity.Ready) ||
		(bb.currentState == connectivity.TransientFailure) != (oldAggrState == connectivity.TransientFailure) {
		bb.regeneratePicker()
	}

	bb.currentConn.UpdateBalancerState(bb.currentState, bb.Picker)
	return
}

func (bb *baseBalancer) regeneratePicker() {
	if bb.currentState == connectivity.TransientFailure {
		log.Info(
			"generated transient error picker",
			logs.String("balancer-name", bb.name),
			logs.String("policy", bb.policy.String()),
		)
		bb.Picker = picker.NewErr(balancer.ErrNoSubConnAvailable)
		return
	}

	// only pass ready subconns to picker
	addrToSc := make(map[resolver.Address]balancer.SubConn)
	for sc, addr := range bb.scToAddr {
		if st, ok := bb.scToSt[sc]; ok && st == connectivity.Ready {
			addrToSc[addr] = sc
		}
	}

	switch bb.policy {
	case picker.RoundrobinBalanced:
		bb.Picker = picker.NewRoundrobinBalanced(addrToSc)
	case picker.WeightedRoundrobinBalanced:
		bb.Picker = picker.NewWRRBalanced(addrToSc)
	default:
		panic(fmt.Errorf("invalid balancer picker policy (%d)", bb.policy))
	}

	log.Info(
		"generated picker",
		logs.String("policy", bb.policy.String()),
		logs.Int32("subconn-size", int32(len(addrToSc))),
	)
}

// Close implements "grpc/balancer.Balancer" interface.
// Close is a nop because base balancer doesn't have internal state to clean up,
// and it doesn't need to call RemoveSubConn for the SubConns.
func (bb *baseBalancer) Close() {
	// TODO
}

func init() {
	log.Debug("init add rr && wrr")
	RegisterBuilder(picker.RoundrobinBalanced)
	RegisterBuilder(picker.WeightedRoundrobinBalanced)
}
