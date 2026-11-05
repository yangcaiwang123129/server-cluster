package logic

import (
	"context"
	"slices"

	"github.com/dobyte/due/v2/cluster"
	"github.com/dobyte/due/v2/registry"
	"github.com/dobyte/due/v2/transport"

	"github.com/dobyte/due/v2/codes"
	"server-cluster/internal/rpc"
	"server-cluster/internal/svc"
	"server-cluster/share/pb"
)

// gmCall 调用 game 服的 GM 内部 RPC（method 传 FullMethodName，服务名固定为 GameAdminService）。
func gmCall[Resp any](ctx context.Context, playerID int64, method string, req any) (*Resp, error) {
	client, err := gmClient(ctx, playerID)
	if err != nil {
		return nil, err
	}
	return rpc.Call[Resp](ctx, client, method, req)
}

// gmClient 在线玩家定向到其所在 game 实例，离线玩家任选一个实例（按需从库加载）。
func gmClient(ctx context.Context, playerID int64) (transport.Client, error) {
	instances, err := gameInstances(ctx, pb.GameAdminService_ServiceDesc.ServiceName)
	if err != nil {
		return nil, err
	}
	if len(instances) == 0 {
		return nil, codes.NewCode(500, "game service unavailable").Err()
	}
	if nodeID := onlineNodeID(ctx, playerID); nodeID != "" {
		return rpc.NewDirectClient(nodeID)
	}
	return rpc.NewDirectClient(instances[0].ID)
}

// onlineNodeID 查询玩家所在的 game 实例（并发拉取全部实例在线列表匹配）；玩家离线返回空串。
func onlineNodeID(ctx context.Context, playerID int64) string {
	if playerID <= 0 {
		return ""
	}
	instances, err := gameInstances(ctx, pb.GameAdminService_ServiceDesc.ServiceName)
	if err != nil {
		return ""
	}
	for _, r := range rpc.CallAll[pb.GetOnlinePlayersResp](ctx, instanceIDs(instances),
		pb.GameAdminService_GetOnlinePlayers_FullMethodName, &pb.GetOnlinePlayersReq{}) {
		if r.Err != nil {
			continue
		}
		for _, p := range r.Resp.Players {
			if p.PlayerId == playerID {
				return p.NodeId
			}
		}
	}
	return ""
}

// gameInstances 枚举提供指定内部服务的全部实例（mesh 实例按 services 字段过滤）。
func gameInstances(ctx context.Context, serviceName string) ([]*registry.ServiceInstance, error) {
	instances, err := svc.Registry().Services(ctx, cluster.Mesh.String())
	if err != nil {
		return nil, err
	}
	matched := make([]*registry.ServiceInstance, 0, len(instances))
	for _, ins := range instances {
		if slices.Contains(ins.Services, serviceName) {
			matched = append(matched, ins)
		}
	}
	return matched, nil
}

// instanceIDs 提取实例 ID 列表（供多实例扇出调用使用）。
func instanceIDs(instances []*registry.ServiceInstance) []string {
	ids := make([]string, 0, len(instances))
	for _, ins := range instances {
		ids = append(ids, ins.ID)
	}
	return ids
}
