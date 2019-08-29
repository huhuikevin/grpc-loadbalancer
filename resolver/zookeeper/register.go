package zookeeper

import (
	"context"
	"encoding/json"
	"github.com/huhuikevin/grpc-loadbalancer/logs"
	"github.com/huhuikevin/grpc-loadbalancer/resolver"
	"github.com/samuel/go-zookeeper/zk"
	uuid "github.com/satori/go.uuid"
	"time"
)

var (
	// Within the session timeout it's possible to reestablish a connection to a different
	// server and keep the same session. This is means any ephemeral nodes and
	// watches are maintained.
	sessionTimeout = time.Duration(time.Second * 5)
    log = logs.SLog
)
//registry register the servers
type registry struct {
	zkClient   *zk.Conn
	event      <-chan zk.Event
	key        string
	serverName string
	value      string
	ctx        context.Context
	cancel     context.CancelFunc
	nodeID     string
}

func init() {
	log.Debug("zk register init")
	resolver.AddRegisterFunc(resolver.ResolverZookeeper, NewRegistry)
}

//NewRegistry register serveice
func NewRegistry(endpoints []string, ServiceName string, option resolver.Option) (resolver.Register, error) {
	zkConn, chanEvent, err := zk.Connect(endpoints, sessionTimeout, zk.WithHostProvider(&DNSHostProvider{}))
	if err != nil {
		log.Error("zk connect:", err)
		return nil, err
	}
	val, err := json.Marshal(option.NData)
	if err != nil {
		return nil, err
	}
	u1 := uuid.NewV4()
	uuid := u1.String()
	if option.NodeID != "" {
		uuid = option.NodeID
	}
	ctx, cancel := context.WithCancel(context.Background())
	r := &registry{
		zkClient:   zkConn,
		event:      chanEvent,
		ctx:        ctx,
		cancel:     cancel,
		serverName: ServiceName,
		value:      string(val),
		key:        resolver.ZkDir + "/" + ServiceName + "/" + uuid,
		nodeID:     uuid,
	}
	return r, nil
}

func (r *registry) checkPathAndCreate(path string, ephemeral bool) error {
	exist, _, err := r.zkClient.Exists(path)
	if err != nil {
		log.Error("check key error:", err)
		return err
	}
	if !exist {
		var flag int32 = 0
		if ephemeral {
			flag = zk.FlagEphemeral
		}
		_, err := r.zkClient.Create(path, nil, flag, zk.WorldACL(zk.PermAll))
		if err != nil {
			log.Error("create error:", err)
			return err
		}
	}
	return nil
}

func (r *registry) Register() error {
	//create root path
	if err := r.checkPathAndCreate(resolver.ZkDir, false); err != nil {
		return err
	}
	if err := r.checkPathAndCreate(resolver.ZkDir + "/" + r.serverName, false); err != nil {
		return err
	}
	s, err := r.zkClient.Create(r.key, []byte(r.value), zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
	log.Debug("create s = ", s)
	if err != nil {
		log.Error("create key error:", err)
		return err
	}
	go func() {
		for event := range r.event {
			if event.Type == zk.EventSession && event.State == zk.StateHasSession {
				exist, _, err := r.zkClient.Exists(r.key)
				if err != nil {
					log.Error("Exists error:", err)
				}
				if !exist {
					r.zkClient.Create(r.key, []byte(r.value), zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
				}
			}
		}
	}()
	return err
}

func (r *registry) Deregister() error {
	r.zkClient.Delete(r.key, 0)
	r.zkClient.Close()
	return nil
}

func (r *registry) NodeID() string {
	return r.nodeID
}
