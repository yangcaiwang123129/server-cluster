package logic

import (
	"context"
	"strings"
	"sync"

	"github.com/dobyte/due/v2/registry"
	"github.com/dobyte/due/v2/transport"

	"server-cluster/internal/rpc"
	"server-cluster/share/errs"
	"server-cluster/share/pb"
)

// gmCall 调用 game 服的 GM 内部 RPC（服务名固定为 GameAdminService）。
// method 传 FullMethodName（/pkg.Service/Method），这里截取末段方法名供 client.Call 拼接。
func gmCall(ctx context.Context, playerID int64, method string, req, resp any) error {
	client, err := gmClient(ctx, playerID)
	if err != nil {
		return err
	}
	return client.Call(ctx, pb.GameAdminService_ServiceDesc.ServiceName, methodName(method), req, resp)
}

// methodName 从 FullMethodName 提取纯方法名。
func methodName(fullMethod string) string {
	if i := strings.LastIndex(fullMethod, "/"); i >= 0 {
		return fullMethod[i+1:]
	}
	return fullMethod
}

// gmClient 选择处理 GM 请求的 game 实例：
// 在线玩家定向到其所在实例（保证命中该玩家正在使用的内存态数据），离线玩家任选一个实例（按需从库加载）。
func gmClient(ctx context.Context, playerID int64) (transport.Client, error) {
	instances, err := gameInstances(ctx, pb.GameAdminService_ServiceDesc.ServiceName)
	if err != nil {
		return nil, err
	}
	if len(instances) == 0 {
		return nil, errs.NewError(500, "game service unavailable")
	}
	if nodeID := onlineNodeID(ctx, playerID); nodeID != "" {
		return rpc.NewDirectClient(nodeID)
	}
	return rpc.NewDirectClient(instances[0].ID)
}

// onlineNodeID 查询玩家所在的 game 实例；玩家离线返回空串。
// 各实例只知道自己的在线玩家，故并发拉取全部实例的在线列表后取首个匹配。
func onlineNodeID(ctx context.Context, playerID int64) string {
	if playerID <= 0 {
		return ""
	}
	instances, err := gameInstances(ctx, pb.GameAdminService_ServiceDesc.ServiceName)
	if err != nil {
		return ""
	}

	var (
		once   sync.Once
		wg     sync.WaitGroup
		nodeID string
	)
	for _, ins := range instances {
		wg.Add(1)
		go func(ins *registry.ServiceInstance) {
			defer wg.Done()
			client, err := rpc.NewDirectClient(ins.ID)
			if err != nil {
				return
			}
			resp := &pb.GetOnlinePlayersResp{}
			if err := client.Call(ctx, pb.GameAdminService_ServiceDesc.ServiceName, methodName(pb.GameAdminService_GetOnlinePlayers_FullMethodName), &pb.GetOnlinePlayersReq{}, resp); err != nil {
				return
			}
			for _, p := range resp.Players {
				if p.PlayerId == playerID {
					once.Do(func() { nodeID = p.NodeId })
					return
				}
			}
		}(ins)
	}
	wg.Wait()
	return nodeID
}
