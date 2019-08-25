package rpcserver

import (
	"context"
	"log"
	"net"
	"sync"
	"time"

	"github.com/huhuikevin/grpc-loadbalancer/resolver"

	"google.golang.org/grpc"
)

const (
	defaultTTL = 10 * time.Second
)


//Config server config
type Config struct {
	Resolver   string
	Domain     string
	Address    string
	ExtAddress string
	Weight     int32
	Disable    int32
	TTL        time.Duration
}

//Server 对具体的rpc server的封装
type Server struct {
	//server     GRPCServer
	grpcserver *grpc.Server
	register   resolver.Register
	config     Config
}

//New 创建一个新的grpc server
func New(config Config) *Server {
	if config.TTL == 0 {
		config.TTL = defaultTTL
	}
	server := &Server{
		config: config,
		grpcserver: grpc.NewServer(),
	}
	return server
}

//NodeID 获取服务注册的节点号
func (s *Server) NodeID() string {
	if s.register == nil {
		return ""
	}
	return s.register.NodeID()
}

func (s *Server)GetgRpcServer() *grpc.Server {
	return s.grpcserver
}

//Start gprc server
func (s *Server) Start(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.config.Address)
	if err != nil {
		log.Printf("failed to listen: %v", err)
		return err
	}
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer listener.Close()
		defer s.grpcserver.Stop()
		wg.Done()
		log.Println("start grpc server")
		s.grpcserver.Serve(listener)
	}()
	wg.Wait()
	register, err := resolver.NewRegister(s.config.Resolver, s.config.Domain, resolver.Option{
		NData: resolver.NodeData{
			Addr:     s.config.ExtAddress,
			MetaData: resolver.BalanceData{Weight: s.config.Weight, Disable: s.config.Disable},
		},
		TTL: s.config.TTL,
	})
	if err != nil {
		log.Println("regist error=", err)
		return err
	}
	s.register = register

	s.register.Register()

	//go func() {
	<- ctx.Done()
	s.Stop()
	//}()
	return nil
}

//Stop stop the server
func (s *Server) Stop() {
	s.register.Deregister()
	s.grpcserver.GracefulStop()
}

