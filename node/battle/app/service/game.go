// Package service 跨服务 mesh 客户端初始化（占位：后续接入 game 等 mesh 服务时在此创建 client）。
package service

import "github.com/dobyte/due/v2/cluster/node"

var proxy *node.Proxy

// Init 初始化 mesh 服务客户端；当前无具体调用，保留 proxy 引用供后续扩展。
func Init(p *node.Proxy) error {
	proxy = p
	return nil
}
