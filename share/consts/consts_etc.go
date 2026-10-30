package consts

import "github.com/dobyte/due/v2/etc"

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
