package resolver

import (
	"fmt"

	"github.com/huhuikevin/grpc-loadbalancer/logs"

	grpcresolver "google.golang.org/grpc/resolver"
)

var log = logs.SLog

const (
	normalBackEndSize = 20
)

type resolverBuilder struct {
	scheme string
}

func (*resolverBuilder) Build(target grpcresolver.Target, cc grpcresolver.ClientConn, opts grpcresolver.BuildOption) (grpcresolver.Resolver, error) {
	log.Debug("build resolver", target.Scheme)
	if !NameServerIsValide(target.Scheme) {
		return nil, fmt.Errorf("scheme:%s is not support", target.Scheme)
	}
	datachan := make(chan []ResolvedData)
	quit := make(chan int)
	endpoints := GetNameServers(target.Scheme)
	if endpoints == nil {
		return nil, fmt.Errorf("scheme:%s have not any endpoints", target.Scheme)
	}
	watcher, err := NewWatcher(target.Scheme, target.Endpoint, endpoints)
	if err != nil {
		log.Error("get watch:", err)
		return nil, err
	}
	r := &GRPCResolver{
		target:   target,
		cc:       cc,
		dataChan: datachan,
		quit:     quit,
		watcher:  watcher,
		address:  make([]grpcresolver.Address, 0, normalBackEndSize),
	}

	go r.start()

	return r, nil
}
func (rb *resolverBuilder) Scheme() string { return rb.scheme }

// GRPCResolver is a
// GRPCResolver (https://godoc.org/google.golang.org/grpc/resolver#Resolver).
type GRPCResolver struct {
	target   grpcresolver.Target
	cc       grpcresolver.ClientConn
	watcher  Watcher
	dataChan chan []ResolvedData
	quit     chan int
	address  []grpcresolver.Address
}

func (r *GRPCResolver) start() {
	r.watcher.Start(r.dataChan)

	for {
		select {
		case rdata := <-r.dataChan:
			updates := make([]grpcresolver.Address, 0, len(rdata))
			for _, data := range rdata {
				resolvedAddr := grpcresolver.Address{
					Addr:     data.Addr,
					Metadata: data.MetaData,
					Type:     grpcresolver.Backend,
				}
				log.Info(logs.String("Resolver", r.target.Scheme), logs.String("address", data.Addr))
				updates = append(updates, resolvedAddr)
			}
			r.cc.UpdateState(grpcresolver.State{Addresses: updates})
			log.Info("update ok")
		case <-r.quit:
			log.Info("Resolver Closed")
			r.watcher.Close()
			return
		}
	}
}

//ResolveNow implemetion the gprc Resolve
func (*GRPCResolver) ResolveNow(o grpcresolver.ResolveNowOption) {
}

//Close implemetion the gprc Resolve
func (r *GRPCResolver) Close() {
	log.Info("Resolver Close...")
	r.quit <- 1
}

func init() {
	//注册ETCD3的域名解析系统到grpc，目前只实现了etcd3，后期可以实现consul，zk等
	log.Debug("init register resolver")
	grpcresolver.Register(&resolverBuilder{ResolverETCD3})
	grpcresolver.Register(&resolverBuilder{ResolverZookeeper})
}
