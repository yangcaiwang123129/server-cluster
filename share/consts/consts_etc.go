package consts

import (
	"net"
	"strings"

	"github.com/dobyte/due/v2/cluster"
	"github.com/dobyte/due/v2/etc"
	"github.com/dobyte/due/v2/registry"
)

// GateInstanceID 网关服务器实例id
func GateInstanceID() string {
	if id := etc.Get("etc.cluster.gate.id").String(); id != "" {
		return id
	}
	return ""
}

// NodeInstanceID 节点服务器实例id
func NodeInstanceID() string {
	if id := etc.Get("etc.cluster.node.id").String(); id != "" {
		return id
	}
	return ""
}

// MeshInstanceID 节点服务器实例id
func MeshInstanceID() string {
	if id := etc.Get("etc.cluster.mesh.id").String(); id != "" {
		return id
	}
	return ""
}

// MeshName 节点服务器别名
func MeshName() string {
	if name := etc.Get("etc.cluster.mesh.name").String(); name != "" {
		return name
	}
	return ""
}

// ExtractHTTPAddr 从 etc.http.addr 提取 host:port（给 mesh 注册用 endpoint）。
func ExtractHTTPAddr() string {
	addr := etc.Get("etc.http.addr").String()
	if addr == "" {
		return ""
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	if host == "" || host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	return host + ":" + port
}

// BuildMeshInstance 构造一个 mesh kind 的 ServiceInstance（name 须为集群服务名，alias 由 id 前缀推导）。
func BuildMeshInstance(id, name, endpoint string) *registry.ServiceInstance {
	if id == "" {
		return nil
	}
	alias := name
	if i := strings.LastIndex(id, "-"); i > 0 && i+1 < len(id) {
		alias = id[:i] // admin-1 → admin, login-1 → login
	}
	return &registry.ServiceInstance{
		ID:       id,
		Name:     name,
		Kind:     cluster.Mesh.String(),
		Alias:    alias,
		State:    "work",
		Endpoint: "http://" + endpoint,
		Weight:   1,
	}
}
