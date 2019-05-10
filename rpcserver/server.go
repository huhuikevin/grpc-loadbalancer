package rpcserver

import (
	"log"
	"net"
	"sync"

	"github.com/huhuikevin/grpc-loadbalancer/resolver"

	"google.golang.org/grpc"
)

//GRPCServer any grpc server must implemente this interface
//提供grpc的server必须要实现这个接口
type GRPCServer interface {
	Register(*grpc.Server)
}

//Config server config
type Config struct {
	Resolver   string
	Domain     string
	Address    string
	ExtAddress string
	Weight     int32
	Disable    int32
}

//Server 对具体的rpc server的封装
type Server struct {
	server     GRPCServer
	grpcserver *grpc.Server
	register   resolver.Register
	config     Config
}

//New 创建一个新的grpc server
func New(config Config) *Server {
	server := &Server{
		config: config,
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

//Start gprc server
func (s *Server) Start(server GRPCServer) error {
	listener, err := net.Listen("tcp", s.config.Address)
	if err != nil {
		log.Printf("failed to listen: %v", err)
		return err
	}
	s.grpcserver = grpc.NewServer()
	s.server = server
	s.server.Register(s.grpcserver)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer listener.Close()
		defer s.grpcserver.Stop()

		if err := s.grpcserver.Serve(listener); err != nil {
			return
		}
		return
	}()

	register, err := resolver.NewRegister(s.config.Resolver, s.config.Domain, resolver.Option{
		NData: resolver.NodeData{
			Addr:     s.config.ExtAddress,
			MetaData: resolver.BalanceData{Weight: s.config.Weight, Disable: s.config.Disable},
		},
		TTL: 9,
	})
	if err != nil {
		return err
	}
	s.register = register
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer s.register.Deregister()
		s.register.Register()
	}()

	wg.Wait()

	return nil
}

//Stop stop the server
func (s *Server) Stop() {
	s.register.Deregister()
	s.grpcserver.GracefulStop()
}
