#grpc-loadbalancer
    dirs:
        -- balancer: the balancer plugin for grpc, there are two policy:rr && wrr
        -- resolver: the name resolver plugin for grpc, by now only support etcdv3
        -- logs: the log interface which can be implement by users
        -- rpcclient: the wrapper for rpc client which hide some details infos, 
                      and implement a goroutine pool
        -- rpcserver: the wrapper for rpc server, which will start the server 
                      and register it on etcd3
        -- example: client&server examples for using load balancer and name resovler
        
    depends:
        -- grpc, version=1.21.0-dev
        -- etcd, version=3.3.10