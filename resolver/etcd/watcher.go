package etcd

import (
	"encoding/json"
	"sync"

	"github.com/huhuikevin/grpc-loadbalancer/logs"
	"github.com/huhuikevin/grpc-loadbalancer/resolver"

	etcd3 "github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"golang.org/x/net/context"
)

var log = logs.SLog

// Watcher is the implementation of grpc.naming.Watcher
type Watcher struct {
	key      string
	client   *etcd3.Client
	ctx      context.Context
	cancel   context.CancelFunc
	dataChan chan []resolver.ResolvedData
	lock     sync.Mutex
	close    bool
	address  map[string]resolver.ResolvedData
}

//Close close the watcher
func (w *Watcher) Close() {
	w.lock.Lock()
	defer w.lock.Unlock()
	w.close = true
	w.cancel()
	if w.dataChan != nil {
		close(w.dataChan)
	}
}

func init() {
	resolver.AddWatchFunc(resolver.ResolverETCD3, NewEtcd3Watcher)
}

//NewEtcd3Watcher get new etcd3 watcher
func NewEtcd3Watcher(key string, endpoints []string) (resolver.Watcher, error) {
	config := etcd3.Config{
		Endpoints: endpoints,
	}
	client, err := etcd3.New(config)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	watcher := Watcher{
		key:     resolver.EtcdDir + "/" + key,
		client:  client,
		ctx:     ctx,
		cancel:  cancel,
		lock:    sync.Mutex{},
		address: make(map[string]resolver.ResolvedData),
	}
	return &watcher, nil
}

func (w *Watcher) delResolvedAddress(addres resolver.ResolvedData) {
	log.Info("delete ResolvedAddress", addres.Addr)
	w.lock.Lock()
	defer w.lock.Unlock()
	if w.close {
		return
	}
	if _, ok := w.address[addres.Addr]; ok {
		delete(w.address, addres.Addr)
		w.sendData()
	}
}

func (w *Watcher) delResolvedAddressByKey(key string) {
	log.Info("delete ResolvedAddress key ", key)
	w.lock.Lock()
	defer w.lock.Unlock()
	if w.close {
		return
	}
	for _, v := range w.address {
		if key == v.Key {
			delete(w.address, v.Addr)
			w.sendData()
			break
		}
	}
}

func (w *Watcher) addResolvedAddrs(address []resolver.ResolvedData) {
	w.lock.Lock()
	defer w.lock.Unlock()
	if w.close {
		return
	}
	added := false
	for _, data := range address {
		if ev, ok := w.address[data.Addr]; !ok {
			added = true
			w.address[data.Addr] = data
			log.Info("add ResolvedAddress:", data.Addr, "key:", data.Key)
		} else {
			if !data.Equls(ev) {
				added = true
				w.address[data.Addr] = data
				log.Info("add ResolvedAddress", data.Addr, "key:", data.Key)
			}
		}
	}
	if added {
		w.sendData()
	}
}

func (w *Watcher) sendData() {
	all := make([]resolver.ResolvedData, 0, len(w.address))
	for _, v := range w.address {
		all = append(all, v)
	}
	w.dataChan <- all
}

//Start start the wather
func (w *Watcher) Start(dataChan chan []resolver.ResolvedData) error{
	w.dataChan = dataChan
	go w.start(dataChan)
	return nil
}

func (w *Watcher) start(dataChan chan []resolver.ResolvedData) {
	resp, err := w.client.Get(w.ctx, w.key, etcd3.WithPrefix())
	if err == nil {
		addrs := extractAddrs(resp)
		if len(addrs) > 0 {
			w.addResolvedAddrs(addrs)
		} else {
			log.Warn("Etcd Watcher Get key empty")
		}
	} else {
		log.Error("Etcd Watcher Get key error:", logs.Error(err))
	}
	// generate etcd Watcher
	rch := w.client.Watch(w.ctx, w.key, etcd3.WithPrefix())
	for wresp := range rch {
		for _, ev := range wresp.Events {
			switch ev.Type {
			case mvccpb.PUT:
				nodeData := resolver.ResolvedData{}
				err := json.Unmarshal([]byte(ev.Kv.Value), &nodeData)
				if err != nil {
					log.Error("Parse node data error:", logs.Error(err))
					continue
				}
				if ev.Kv.Key != nil {
					nodeData.Key = string(ev.Kv.Key)
				} else {
					log.Warn("PUT event, resolved data, key is nil")
				}

				w.addResolvedAddrs([]resolver.ResolvedData{nodeData})
			case mvccpb.DELETE:
				//按照etcd3协议，delete的时候，只返回被删除的key
				if ev.Kv.Key == nil {
					log.Error("Delete key is nil")
					continue
				}
				w.delResolvedAddressByKey(string(ev.Kv.Key))
				// nodeData := resolver.ResolvedData{}
				// err := json.Unmarshal([]byte(ev.Kv.Value), &nodeData)
				// if err != nil {
				// 	log.Error("Parse node data error:", logs.Error(err))
				// 	continue
				// }
				// w.delResolvedAddress(nodeData)
			}
		}
	}
}

func extractAddrs(resp *etcd3.GetResponse) []resolver.ResolvedData {
	addrs := []resolver.ResolvedData{}

	if resp == nil || resp.Kvs == nil {
		return addrs
	}

	for i := range resp.Kvs {
		if v := resp.Kvs[i].Value; v != nil {
			nodeData := resolver.ResolvedData{}
			err := json.Unmarshal(v, &nodeData)
			if err != nil {
				log.Error("Parse node data error:", logs.Error(err))
				continue
			}
			if resp.Kvs[i].Key != nil {
				nodeData.Key = string(resp.Kvs[i].Key)
			} else {
				log.Warn("Get resolved data, key is nil")
			}

			addrs = append(addrs, nodeData)
		}
	}

	return addrs
}
