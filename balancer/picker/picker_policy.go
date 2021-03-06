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

import "fmt"

// Policy defines balancer picker policy.
type Policy uint8

const (
	// RoundrobinBalanced balance loads over multiple endpoints
	// and implements failover in roundrobin fashion.
	RoundrobinBalanced Policy = iota

	//RoundrobinBalancedInGrpc is the grpc's internal lb
	RoundrobinBalancedInGrpc
	// WeightedRoundrobinBalanced balance loads over multiple endpoints
	// and implements failover in weighted roundrobin
	WeightedRoundrobinBalanced

	// LeastConnectionBalanced balance loads over multiple endpoints
	// and implements failover for choising least connection
	LeastConnectionBalanced
	// TODO: only send loads to pinned address "RoundrobinFailover"
	// just like how 3.3 client works
	//
	// TODO: prioritize leader
	// TODO: health-check
	// TODO: weighted roundrobin
	// TODO: power of two random choice
)

func (p Policy) String() string {
	switch p {
	case RoundrobinBalanced:
		return "roundrobin-balanced"
	case WeightedRoundrobinBalanced:
		return "wroundrobin-balanced"
	case RoundrobinBalancedInGrpc:
		return "round_robin"
	case LeastConnectionBalanced:
		panic(fmt.Errorf("not implemention balancer picker policy (%d)", p))
	default:
		panic(fmt.Errorf("invalid balancer picker policy (%d)", p))
	}
}

//Name policy name
func (p Policy) Name() string {
	return p.String()
}
