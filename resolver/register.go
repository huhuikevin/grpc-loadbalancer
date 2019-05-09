package resolver

import (
	"errors"
	"fmt"
	"time"
)

//Option register option
type Option struct {
	NData  NodeData
	TTL    time.Duration
	NodeID string
}

//NodeData registeed info
type NodeData struct {
	//Addr 服务地址ip:port
	Addr string
	//MetaData 给负载均衡器使用的数据
	//在使用grpc内置的rr lb的时候不用，主要是给自定义lb使用
	MetaData BalanceData
}

//BalanceData 给负载均衡器用来决策的数据
type BalanceData struct {
	//weight 权重, 这个值越大使用的频率也越大
	//在使用weight round robin负载均衡器的时候使用
	Weight int32
	//Disable 关闭这个服务，一般测试的时候使用
	Disable int32
}

//Register 注册接口
type Register interface {
	//Register 开始注册
	Register() error
	//Deregister 注销
	Deregister() error
	//获取注册节点号
	NodeID() string
}

//RegisterFunc register function
type RegisterFunc func(endpoints []string, ServiceName string, option Option) (Register, error)

var register = make(map[string]RegisterFunc)

//NewRegister 获取一个注册类型，通过它可以注册或者注销我们的服务
func NewRegister(name string, server string, opts Option) (Register, error) {
	if name != ResolverETCD3 {
		return nil, fmt.Errorf("NewRegister: name %s is not supported", name)
	}
	endpoints := GetNameServers(name)
	fun, ok := register[name]
	if !ok {
		return nil, errors.New("Can not found Register function")
	}
	return fun(endpoints, server, opts)
}

//AddRegisterFunc add register func for name
func AddRegisterFunc(name string, f RegisterFunc) {
	register[name] = f
}
