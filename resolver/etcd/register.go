package etcd

import (
	"encoding/json"
	"time"

	"github.com/huhuikevin/grpc-loadbalancer/logs"
	"github.com/huhuikevin/grpc-loadbalancer/resolver"

	etcd3 "github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/etcdserver/api/v3rpc/rpctypes"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/net/context"
)

//registry register the servers
type registry struct {
	etcd3Client *etcd3.Client
	key         string
	value       string
	ttl         time.Duration
	ctx         context.Context
	cancel      context.CancelFunc
	id          etcd3.LeaseID
	nodeID      string
}

func init() {
	log.Debug("init")
	resolver.AddRegisterFunc(resolver.ResolverETCD3, NewRegistry)
}

//NewRegistry register serveice
func NewRegistry(endpoints []string, ServiceName string, option resolver.Option) (resolver.Register, error) {
	log.Debug("NewRegistry", ServiceName)
	config := etcd3.Config{
		Endpoints: endpoints,
	}
	client, err := etcd3.New(config)
	if err != nil {
		return nil, err
	}

	val, err := json.Marshal(option.NData)
	if err != nil {
		return nil, err
	}
	u1, err := uuid.NewV4()
	if err != nil {
		u1, err = uuid.NewV1()
		if err != nil {
			return nil, err
		}
	}
	uuid := u1.String()
	ctx, cancel := context.WithCancel(context.Background())
	registry := &registry{
		etcd3Client: client,
		key:         resolver.EtcdDir + "/" + ServiceName + "/" + uuid,
		value:       string(val),
		ttl:         option.TTL,
		ctx:         ctx,
		cancel:      cancel,
		nodeID:      uuid,
	}

	return registry, nil
}

//NodeID 获取nodeid
func (e *registry) NodeID() string {
	return e.nodeID
}

//Register begin register
func (e *registry) Register() error {
	log.Debug("start register", logs.Int64("ttl=", int64(e.ttl)))
	insertFunc := func() error {
		if e.id == 0 {
			resp, err := e.etcd3Client.Grant(e.ctx, int64(e.ttl))
			if err != nil {
				return err
			}
			e.id = resp.ID
		}
		_, err := e.etcd3Client.Get(e.ctx, e.key)
		if err != nil {
			if err == rpctypes.ErrKeyNotFound {
				if _, err := e.etcd3Client.Put(e.ctx, e.key, e.value, etcd3.WithLease(e.id)); err != nil {
					log.Error("grpclb: set key with ttl to etcd3 failed:", e.key, err.Error())
				}
			} else {
				log.Error("grpclb: key connect to etcd3 failed", e.key, err.Error())
			}
			return err
		}
		// refresh set to true for not notifying the watcher
		if _, err := e.etcd3Client.Put(e.ctx, e.key, e.value, etcd3.WithLease(e.id)); err != nil {
			log.Error("grpclb: refresh key with ttl to etcd3 failed", e.key, err.Error())
			return err
		}

		return nil
	}

	err := insertFunc()
	if err != nil {
		return err
	}
	go func() {
		for {
			resp, err := e.etcd3Client.KeepAlive(e.ctx, e.id)
			if err != nil {
				log.Error("keepalive error", err.Error())
			} else {
				data := <-resp
				if data == nil {
					e.id = 0
					go e.Register()
					return
				}
				log.Info(logs.Int64("TTL=", data.TTL))
			}
		}
	}()
	select {
	case <-e.ctx.Done():
		if _, err := e.etcd3Client.Delete(context.Background(), e.key); err != nil {
			log.Error("grpclb: deregister failed:", e.key, err.Error())
		}
		return nil
	}

	// ticker := time.NewTicker(e.ttl * time.Second / 5)
	// for {
	// 	select {
	// 	case <-ticker.C:
	// 		insertFunc()
	// 	case <-e.ctx.Done():
	// 		ticker.Stop()
	// 		if _, err := e.etcd3Client.Delete(context.Background(), e.key); err != nil {
	// 			log.Error("grpclb: deregister failed:", e.key, err.Error())
	// 		}
	// 		return nil
	// 	}
	// }
}

//Deregister unregister
func (e *registry) Deregister() error {
	e.cancel()
	return nil
}
