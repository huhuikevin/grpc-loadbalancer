package resolver

const (
	//ResolverETCD3 利用etcd3做域名解析
	ResolverETCD3 = "etcd3"
	//EtcdDir 利用etcd3做域名解析的时候，使用的父目录
	EtcdDir = "/grpc-naming"

	//ResolverConsul 利用consul做域名解析
	ResolverConsul = "consul"

	//ResolverZookeeper 利用zoopeeker做域名解析
	ResolverZookeeper = "zoopeeker"
)

var resolverServers = make(map[string][]string)

//AddNameServers 设置域名解析服务器地址
func AddNameServers(name string, servers []string) {
	if name != ResolverETCD3 {
		log.Error("server", name, "is Not support!!")
		return
	}
	resolverServers[name] = servers
}

//GetNameServers 获取域名解析服务器地址
func GetNameServers(name string) []string {
	if name != ResolverETCD3 {
		log.Error("server", name, "is Not support!!")
		return nil
	}
	return resolverServers[name]
}

//NameServerIsValide 给定的name是否支持
func NameServerIsValide(name string) bool {
	if name != ResolverETCD3 {
		log.Error("server", name, "is Not support!!")
		return false
	}
	return true
}
