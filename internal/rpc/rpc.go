// Package rpc 服务间 RPC 调用封装：基于 HTTP 组件代理创建 mesh 客户端，并提供并发批调用。
package rpc

import (
	"context"
	"sync"

	"server-cluster/internal/svc"

	"github.com/dobyte/due/v2/transport"
)

// discoveryScheme mesh 服务发现地址前缀；其后拼接 provider 服务名。
const discoveryScheme = "discovery://"

// directScheme mesh 定向直连地址前缀；其后拼接实例 ID 或地址。
const directScheme = "direct://"

// NewClient 创建指向指定 provider 服务名的 mesh 客户端。
// target 须为服务名（discovery 按实例的 services 字段分组），而非节点名。
func NewClient(service string) (transport.Client, error) {
	return svc.HTTPProxy().NewMeshClient(discoveryScheme + service)
}

// NewDirectClient 创建定向到指定实例的 mesh 客户端（direct://<instance_id>）。
// 实例 ID → 地址的解析由传输器从注册发现缓存的实例集中完成。
func NewDirectClient(instanceID string) (transport.Client, error) {
	return svc.HTTPProxy().NewMeshClient(directScheme + instanceID)
}

// Invoke 发起一次服务间调用；连接由 transporter 连接池复用，重复创建客户端开销低。
func Invoke(ctx context.Context, service, method string, req, resp any) error {
	client, err := NewClient(service)
	if err != nil {
		return err
	}
	return client.Call(ctx, service, method, req, resp)
}

// Call 一次并发调用的描述。
type Call struct {
	Service string
	Method  string
	Req     any
	Resp    any
}

// InvokeAll 并发执行多次调用，返回与 calls 顺序一致的错误切片（nil 表示成功）。
func InvokeAll(ctx context.Context, calls ...Call) []error {
	errs := make([]error, len(calls))
	var wg sync.WaitGroup
	wg.Add(len(calls))
	for i := range calls {
		go func(i int) {
			defer wg.Done()
			c := calls[i]
			errs[i] = Invoke(ctx, c.Service, c.Method, c.Req, c.Resp)
		}(i)
	}
	wg.Wait()
	return errs
}
