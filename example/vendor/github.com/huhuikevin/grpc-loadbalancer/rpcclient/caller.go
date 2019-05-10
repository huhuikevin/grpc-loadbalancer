package rpcclient

import (
	"context"
	"errors"
	"reflect"
	"time"

	"google.golang.org/grpc"
)

var (
	//ErrCanNotFoundFunc 想要调用的函数没有找到
	ErrCanNotFoundFunc = errors.New("Can not found the function")
	//ErrReturnValueNotValid 返回的参数不合法
	ErrReturnValueNotValid = errors.New("Number of the return value is Not 2")
	//ErrReturnErrorNotValid 返回的第二个参数不能转为error类型
	ErrReturnErrorNotValid = errors.New("2th returned value Can Not conver to error")
	//ErrReturnValueCanNotConvertToStruct 第一个返回值不能转为正确的struct
	ErrReturnValueCanNotConvertToStruct = errors.New("The returned value can not be convert to struct")
)

//Caller call grpc client method
type Caller struct {
	Resolver   string
	Domain     string
	Balance    string
	Client     interface{}
	Method     map[string]reflect.Value
	workQueue  *workQueue
	grpcClient *grpc.ClientConn
}

//NewCaller 返回一个client 的call对象
func NewCaller(resolver string, domain string, balance string, queueSize int32) *Caller {
	caller := &Caller{
		Resolver:  resolver,
		Domain:    domain,
		Balance:   balance,
		Method:    make(map[string]reflect.Value),
		workQueue: newWorkQueue(queueSize),
	}
	return caller
}

//Stop stop the rpc caller
func (c *Caller) Stop() {
	if c.workQueue != nil {
		c.workQueue.stop()
	}
	if c.grpcClient != nil {
		c.grpcClient.Close()
	}
}

//Start get grpc client connection
func (c *Caller) Start(gclient GRPCClient, methodName []string) error {
	//target is for the naming finder,example etcd:///test.example.com
	//the grpc will use the naming server of "etcd" for name resolver
	target := c.Resolver + ":///" + c.Domain
	client, err := grpc.Dial(target, grpc.WithInsecure(), grpc.WithBalancerName(c.Balance))
	if err != nil {
		return err
	}
	c.grpcClient = client
	c.Client = gclient.NewClient(client)
	for _, name := range methodName {
		_, ok := c.Method[name]
		if !ok {
			value := c.getFunctionByName(name)
			if !value.IsValid() {
				client.Close()
				return ErrCanNotFoundFunc
			}
			c.Method[name] = value
		}
	}
	return nil
}

//InvokeWithArgs2 调用Client中对应名称为mName的方法，其实就是封装了grpc的方法
//返回2个值，一个interface， 一个error
func (c *Caller) InvokeWithArgs2(mName string, params []interface{}, timeout time.Duration) (interface{}, error) {
	value, ok := c.Method[mName]
	if !ok {
		return nil, ErrCanNotFoundFunc
	}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(timeout))
	defer cancel()
	args := make([]reflect.Value, 0, len(params))
	args = append(args, reflect.ValueOf(ctx))
	for _, v := range params {
		args = append(args, reflect.ValueOf(v))
	}
	rvalue := c.callFuncOnWorkqueue(value, args) //value.Call(args)

	if len(rvalue) != 2 {
		return nil, ErrReturnValueNotValid
	}
	if !rvalue[1].IsNil() && rvalue[1].IsValid() {
		r1 := rvalue[1].Interface()
		err, ok := r1.(error)
		if !ok {
			return nil, ErrReturnErrorNotValid
		}
		return nil, err
	}
	r0 := rvalue[0].Interface()
	return r0, nil
}

func (c *Caller) callFuncOnWorkqueue(f reflect.Value, args []reflect.Value) []reflect.Value {
	return c.workQueue.executeTask(f, args)
}

func (c *Caller) getFunctionByName(name string) reflect.Value {
	value := reflect.ValueOf(c.Client)
	return value.MethodByName(name)
}
