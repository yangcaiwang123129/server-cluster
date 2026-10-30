package consts

import "github.com/dobyte/due/v2/etc"

// GateInstanceID 网关服务器实例id
func GateInstanceID() string {
	if id := etc.Get("cluster.gate.id").String(); id != "" {
		return id
	}
	return ""
}

// NodeInstanceID 节点服务器实例id
func NodeInstanceID() string {
	if id := etc.Get("cluster.node.id").String(); id != "" {
		return id
	}
	return ""
}

// MeshInstanceID 节点服务器实例id
func MeshInstanceID() string {
	if id := etc.Get("cluster.mesh.id").String(); id != "" {
		return id
	}
	return ""
}

// MeshAlias 节点服务器别名
func MeshAlias() string {
	if id := etc.Get("cluster.mesh.name").String(); id != "" {
		return id
	}
	return ""
}
