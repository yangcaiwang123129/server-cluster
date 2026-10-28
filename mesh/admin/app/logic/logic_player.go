package logic

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/dobyte/due/v2/cluster"
	"github.com/dobyte/due/v2/registry"

	"server-cluster/internal/rpc"
	"server-cluster/internal/svc"
	"server-cluster/mesh/admin/app/repo"
	"server-cluster/share/dto"
	"server-cluster/share/errs"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
)

// rpcTimeout 服务间调用超时。
const rpcTimeout = 3 * time.Second

// PlayerLogic 玩家管理业务接口。
type PlayerLogic interface {
	// PlayerList 玩家分页列表（直查 player_base）
	PlayerList(req dto.PlayerListReq) ([]dto.PlayerResp, int64, error)
	// PlayerOnline 在线玩家列表（经 mesh 调 game 服内部 RPC）
	PlayerOnline() ([]dto.OnlinePlayerResp, error)
	// PlayerKick 踢玩家下线（经 mesh 调 game 服内部 RPC）
	PlayerKick(req dto.PlayerKickReq) error
}

// playerLogicInst 玩家业务单例容器。
var playerLogicInst = singleton.NewContainer[PlayerLogic]("PlayerLogic")

// RegisterPlayerLogic 装配期注册玩家业务实现。
func RegisterPlayerLogic(inst PlayerLogic) { playerLogicInst.Register(inst) }

// Player 获取玩家模块业务单例。
func Player() PlayerLogic { return playerLogicInst.MustGet() }

// playerLogic 玩家管理业务实现。
type playerLogic struct{}

// init 装配期自注册实现。
func init() {
	RegisterPlayerLogic(&playerLogic{})
}

// PlayerList 玩家分页列表（直查 player_base）。
func (this *playerLogic) PlayerList(req dto.PlayerListReq) ([]dto.PlayerResp, int64, error) {
	return repo.R().ListPlayers(repo.PlayerQuery{
		Username: req.Username,
		Order:    req.Order,
	}, req.PageIndex, req.PageSize)
}

// PlayerOnline 在线玩家列表：枚举提供玩家服务的全部 game 实例，并发定向拉取后聚合。
func (this *playerLogic) PlayerOnline() ([]dto.OnlinePlayerResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	instances, err := gameInstances(ctx, pb.GameAdminService_ServiceDesc.ServiceName)
	if err != nil {
		return nil, err
	}

	list := make([]dto.OnlinePlayerResp, 0)
	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)
	for _, ins := range instances {
		wg.Add(1)
		go func(ins *registry.ServiceInstance) {
			defer wg.Done()
			client, err := rpc.NewDirectClient(ins.ID)
			if err != nil {
				return // 实例不可达：跳过，避免单实例故障拖垮整个列表
			}
			resp := &pb.GetOnlinePlayersResp{}
			if err := client.Call(ctx, pb.GameAdminService_ServiceDesc.ServiceName, methodName(pb.GameAdminService_GetOnlinePlayers_FullMethodName), &pb.GetOnlinePlayersReq{}, resp); err != nil {
				return
			}
			mu.Lock()
			for _, p := range resp.Players {
				list = append(list, dto.OnlinePlayerResp{PlayerID: p.PlayerId, EnterTime: p.EnterTime, NodeID: p.NodeId})
			}
			mu.Unlock()
		}(ins)
	}
	wg.Wait()
	return list, nil
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

// PlayerKick 踢玩家下线：按玩家所在实例定向调用（协议按服批量，单踢为一批）。
func (this *playerLogic) PlayerKick(req dto.PlayerKickReq) error {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()
	client, err := rpc.NewDirectClient(req.NodeID)
	if err != nil {
		return err
	}
	resp := &pb.KickPlayerResp{}
	if err := client.Call(ctx, pb.GameAdminService_ServiceDesc.ServiceName, methodName(pb.GameAdminService_KickPlayer_FullMethodName),
		&pb.KickPlayerReq{PlayerIds: []int64{req.PlayerID}}, resp); err != nil {
		return err
	}
	if len(resp.FailedIds) > 0 {
		return errs.NewError(int(pb.ErrCode_ERR_CODE_PLAYER_NOT_FOUND), "player is not online")
	}
	return nil
}
