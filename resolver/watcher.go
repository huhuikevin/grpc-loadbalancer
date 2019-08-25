package resolver

import (
	"errors"
	"fmt"
)

//ResolvedData 返回名称解析后的地址和参数，参数包括了负载均衡器的工作参数
type ResolvedData struct {
	NodeData
	//Key 对应的服务注册的Key
	Key string
}

//Equls 判断两个ResolvedData是否一样
func (r ResolvedData) Equls(d ResolvedData) bool {
	if r.Addr != d.Addr {
		return false
	}
	if r.Key != d.Key {
		return false
	}
	if r.MetaData.Disable != d.MetaData.Disable {
		return false
	}
	if r.MetaData.Weight != d.MetaData.Weight {
		return false
	}
	return true
}

//Watcher etcd， consul的观察者，检测服务的变化
type Watcher interface {
	//Close 函数用来关闭观察者
	Close()
	Start(dataChan chan []ResolvedData) error
}

//WatcherFunc watch function
type WatcherFunc func(name string, endpoints []string) (Watcher, error)

var watcher = make(map[string]WatcherFunc)

//NewWatcher 获取一个观察值
//参数
//name: 使用etcd，consol，zk?
//service: 需要监测的key
//endpoints: 用于域名解析的服务器地址列表
func NewWatcher(name string, service string, endpoints []string) (Watcher, error) {
	if name == ResolverETCD3 || name == ResolverZookeeper{
		fun, ok := watcher[name]
		if !ok {
			return nil, errors.New("can not get Watcher func")
		}
		watcher, err := fun(service, endpoints)
		if err != nil {
			return nil, err
		}
		return watcher, nil
	}
	return nil, fmt.Errorf("NewWatcher: name %s is not supported", name)
}

//AddWatchFunc add watch function for name
func AddWatchFunc(name string, f WatcherFunc) {
	watcher[name] = f
}
