package zookeeper

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/huhuikevin/grpc-loadbalancer/resolver"
	"github.com/samuel/go-zookeeper/zk"
	"sync"
	"time"
)

type watcher struct {
	key        string
	serverName string
	zkClient   *zk.Conn
	event      <-chan zk.Event
	ctx        context.Context
	cancel     context.CancelFunc
	dataChan   chan []resolver.ResolvedData
	lock       sync.Mutex
	close      bool
	address    map[string]resolver.ResolvedData
}

func init() {
	log.Debug("zk watch init")
	resolver.AddWatchFunc(resolver.ResolverZookeeper, NewZkWatcher)
}

func NewZkWatcher(serverName string, endpoints []string) (resolver.Watcher, error) {
	zkConn, chanEvent, err := zk.Connect(endpoints, sessionTimeout, zk.WithHostProvider(&DNSHostProvider{}))
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	watcher := watcher{
		key:        resolver.ZkDir + "/" + serverName,
		serverName: serverName,
		zkClient:   zkConn,
		event:      chanEvent,
		ctx:        ctx,
		cancel:     cancel,
		lock:       sync.Mutex{},
		address:    make(map[string]resolver.ResolvedData),
	}
	return &watcher, nil
}

func (w *watcher) Close() {
	w.cancel()
}


func (w *watcher) Start(dataChan chan []resolver.ResolvedData) error{
	w.dataChan = dataChan
	if err := w.checkPathAndCreate(resolver.ZkDir, false); err != nil {
		return err
	}
	if err := w.checkPathAndCreate(resolver.ZkDir + "/" + w.serverName, false); err != nil {
		return err
	}
	go w.start()
	return nil
}

func (w *watcher) start() {
	for {
		log.Debug("key=", w.key)
		childs, _, echan, err := w.zkClient.ChildrenW(w.key)
		if err != nil {
			fmt.Println("watch error:", err)
			time.Sleep(time.Duration(time.Second))
			continue
		}
		log.Debug("get:", childs)
		w.fetchAddressInfo(childs)
		select {
		case e := <- echan:
			log.Debug("event type=", e.Type.String())
		case <- w.ctx.Done():
			log.Debug("ctx done, exit...")
			break;
		}
	}
	w.zkClient.Close()
	close(w.dataChan)
}

//判断地址是否有变化
func (w *watcher)changeAddress(address[]resolver.ResolvedData) {
	w.lock.Lock()
	defer w.lock.Unlock()

	changed := false
	if len(address) != len(w.address) {
		changed = true
	}else {
		notEqules := 0
		for _, addr := range address {
			d, ok := w.address[addr.Addr]
			if !ok || !d.Equls(addr){
				notEqules ++
				break
			}
		}
		if notEqules > 0 {
			changed = true
		}
	}
	if changed {
		w.address = make(map[string]resolver.ResolvedData)
		for _, addr := range address {
			w.address[addr.Addr] = addr
		}
		w.notifyChanged()
	}
}

func (w *watcher) notifyChanged() {
	all := make([]resolver.ResolvedData, 0, len(w.address))
	addrs := make([]string, 0, len(w.address))
	for a, v := range w.address {
		all = append(all, v)
		addrs = append(addrs, a)
	}
	select {
	case w.dataChan <- all:
		break
	case <- w.ctx.Done():
		return
	}
}

func (w *watcher)fetchAddressInfo(childs[]string) {
	address := make([]resolver.ResolvedData, 0, len(childs))
	for _, child := range childs {
		data, _, err := w.zkClient.Get(w.key + "/" + child)
		if err != nil {
			fmt.Println(err)
			continue
		}
		nodeData := resolver.ResolvedData{}
		err = json.Unmarshal(data, &nodeData)
		if err != nil {
			fmt.Println(err)
			continue
		}
		nodeData.Key = child
		address = append(address, nodeData)
	}
	if len(address) > 0 {
		w.changeAddress(address)
	}
}

func (w *watcher) checkPathAndCreate(path string, ephemeral bool) error {
	exist, _, err := w.zkClient.Exists(path)
	if err != nil {
		fmt.Println("check key error:", err)
		return err
	}
	if !exist {
		var flag int32 = 0
		if ephemeral {
			flag = zk.FlagEphemeral
		}
		_, err := w.zkClient.Create(path, nil, flag, zk.WorldACL(zk.PermAll))
		if err != nil {
			fmt.Println("create error:", err)
			return err
		}
	}
	return nil
}
