// Package rpc 服务间 RPC 调用封装：基于 HTTP 组件代理创建 mesh 客户端，并提供泛型调用入口。
package rpc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"server-cluster/internal/svc"

	"github.com/dobyte/due/v2/transport"
)

// discoveryScheme mesh 服务发现地址前缀；其后拼接 provider 服务名。
const discoveryScheme = "discovery://"

// directScheme mesh 定向直连地址前缀；其后拼接实例 ID 或地址。
const directScheme = "direct://"

// defaultRPCTimeout 调用方 ctx 未携带 deadline 时施加的默认超时。
const defaultRPCTimeout = 3 * time.Second

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

// Call 泛型远程调用：fullMethod 传 FullMethodName（/pkg.Service/Method），
// 自动拆出 gRPC 服务名与短方法名；ctx 未携带 deadline 时施加默认超时；返回强类型响应。
func Call[Resp any](ctx context.Context, client transport.Client, fullMethod string, req any) (*Resp, error) {
	service, method, err := splitFullMethod(fullMethod)
	if err != nil {
		return nil, err
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultRPCTimeout)
		defer cancel()
	}
	resp := new(Resp)
	if err := client.Call(ctx, service, method, req, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// splitFullMethod 将 FullMethodName（/pkg.Service/Method）拆为 gRPC 服务名与短方法名。
func splitFullMethod(fullMethod string) (service, method string, err error) {
	fullMethod = strings.TrimPrefix(fullMethod, "/")
	i := strings.LastIndex(fullMethod, "/")
	if i <= 0 || i == len(fullMethod)-1 {
		return "", "", fmt.Errorf("rpc: invalid full method %q", fullMethod)
	}
	return fullMethod[:i], fullMethod[i+1:], nil
}
