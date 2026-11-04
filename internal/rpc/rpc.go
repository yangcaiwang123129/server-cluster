// Package rpc due transport 泛型 RPC 封装：统一 mesh 客户端创建、超时控制与
// FullMethodName 处理，并提供多实例并发扇出调用。
package rpc

import (
	"context"
	"fmt"
	"strings"
	"sync"
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
// 自动拆出 gRPC 服务名与短方法名；ctx 未携带 deadline 时施加默认超时。
// 调用失败时返回 (nil, err)——gRPC 出错时 reply 可能被部分填充，直接弃用以避免调用方误读。
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

// InstanceResult 单实例调用结果；Resp 仅在 Err 为 nil 时有效。
type InstanceResult[T any] struct {
	InstanceID string
	Resp       *T
	Err        error
}

// CallAll 并发调用全部指定实例；任一实例失败不中断其他实例，错误记录在对应结果中。
func CallAll[T any](ctx context.Context, instanceIDs []string, fullMethod string, req any) []InstanceResult[T] {
	results := make([]InstanceResult[T], len(instanceIDs))
	var wg sync.WaitGroup
	for i, id := range instanceIDs {
		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()
			results[i].InstanceID = id
			client, err := NewDirectClient(id)
			if err != nil {
				results[i].Err = err
				return
			}
			results[i].Resp, results[i].Err = Call[T](ctx, client, fullMethod, req)
		}(i, id)
	}
	wg.Wait()
	return results
}
