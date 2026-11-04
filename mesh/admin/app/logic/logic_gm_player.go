package logic

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"slices"
	"sync"

	"github.com/dobyte/due/v2/cluster"
	"github.com/dobyte/due/v2/registry"

	"server-cluster/internal/rpc"
	"server-cluster/internal/svc"
	"server-cluster/mesh/admin/app/repo"
	"server-cluster/share/dto"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
	"server-cluster/share/util"

	"github.com/dobyte/due/v2/codes"
)

// PlayerLogic 玩家管理业务接口。
type PlayerLogic interface {
	// PlayerList 玩家分页列表（直查 player）
	PlayerList(req dto.PlayerListReq) ([]dto.PlayerResp, int64, error)
	// PlayerOnline 在线玩家列表（经 mesh 调 game 服内部 RPC）
	PlayerOnline() ([]dto.OnlinePlayerResp, error)
	// PlayerKick 踢玩家下线（经 mesh 调 game 服内部 RPC）
	PlayerKick(req dto.PlayerKickReq) error
	// PlayerBan 封号（更新 DB，同时如果在线则踢下线）
	PlayerBan(req dto.PlayerBanReq) error
	// PlayerUnban 解封（恢复状态）
	PlayerUnban(req dto.PlayerUnbanReq) error
	// PlayerResetPassword 重置密码（明文 MD5 哈希后存库）
	PlayerResetPassword(req dto.PlayerResetPasswordReq) error
}

// playerLogicInst 玩家业务单例容器。
var playerLogicInst = singleton.NewContainer[PlayerLogic]("PlayerLogic")

// RegisterPlayerLogic 装配期注册玩家业务实现。
func RegisterPlayerLogic(inst PlayerLogic) { playerLogicInst.Register(inst) }

// Player 获取玩家模块业务单例。
func Player() PlayerLogic { return playerLogicInst.MustGet() }

// playerLogic 玩家管理业务实现。
type playerLogic struct{}

func init() {
	RegisterPlayerLogic(&playerLogic{})
}

// PlayerList 玩家分页列表（直查 player）。
func (this *playerLogic) PlayerList(req dto.PlayerListReq) ([]dto.PlayerResp, int64, error) {
	return repo.R().ListPlayers(repo.PlayerQuery{
		PlayerID:       req.PlayerID,
		Username:       req.Username,
		PhoneNumber:    req.PhoneNumber,
		Status:         req.Status,
		CreatedAtStart: req.CreatedAtStart,
		CreatedAtEnd:   req.CreatedAtEnd,
		Order:          req.Order,
	}, req.PageIndex, req.PageSize)
}

// PlayerOnline 在线玩家列表：枚举提供玩家服务的全部 game 实例，并发定向拉取后聚合。
func (this *playerLogic) PlayerOnline() ([]dto.OnlinePlayerResp, error) {
	instances, err := gameInstances(context.Background(), pb.GameAdminService_ServiceDesc.ServiceName)
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
			resp, err := rpc.Call[pb.GetOnlinePlayersResp](context.Background(), client, pb.GameAdminService_GetOnlinePlayers_FullMethodName, &pb.GetOnlinePlayersReq{})
			if err != nil {
				return
			}
			mu.Lock()
			for _, p := range resp.Players {
				list = append(list, dto.OnlinePlayerResp{
					PlayerID:    p.PlayerId,
					EnterTime:   p.EnterTime,
					LobbyNodeID: p.NodeId,
					LoginIP:     p.LoginIp,
				})
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
	client, err := rpc.NewDirectClient(req.NodeID)
	if err != nil {
		return err
	}
	resp, err := rpc.Call[pb.KickPlayerResp](context.Background(), client, pb.GameAdminService_KickPlayer_FullMethodName,
		&pb.KickPlayerReq{PlayerIds: []int64{req.PlayerID}})
	if err != nil {
		return err
	}
	if len(resp.FailedIds) > 0 {
		return codes.NewCode(int(pb.ErrCode_ERR_CODE_PLAYER_NOT_FOUND), "player is not online").Err()
	}
	return nil
}

// PlayerBan 封号：写 DB + 如果该玩家在线则同时踢下线（KickType=2 封号踢）。
func (this *playerLogic) PlayerBan(req dto.PlayerBanReq) error {
	// 计算到期时间：0 表示永久
	var banUntil int64
	if req.Duration > 0 {
		banUntil = util.NowUnix() + int64(req.Duration)
	}
	if err := repo.R().BanPlayer(req.PlayerID, banUntil, req.Reason); err != nil {
		return err
	}
	// 在线则按服踢下线：枚举 mesh 实例找玩家（简化版，逐实例尝试，失败忽略）
	instances, _ := gameInstances(context.Background(), pb.GameAdminService_ServiceDesc.ServiceName)
	for _, ins := range instances {
		client, err := rpc.NewDirectClient(ins.ID)
		if err != nil {
			continue
		}
		_, _ = rpc.Call[pb.KickPlayerResp](context.Background(), client, pb.GameAdminService_KickPlayer_FullMethodName,
			&pb.KickPlayerReq{PlayerIds: []int64{req.PlayerID}})
	}
	return nil
}

// PlayerUnban 解封。
func (this *playerLogic) PlayerUnban(req dto.PlayerUnbanReq) error {
	return repo.R().UnbanPlayer(req.PlayerID)
}

// PlayerResetPassword 重置密码：明文 MD5 哈希后写库（与 login 服保持一致）。
func (this *playerLogic) PlayerResetPassword(req dto.PlayerResetPasswordReq) error {
	h := md5.Sum([]byte(req.NewPassword))
	hash := hex.EncodeToString(h[:])
	return repo.R().ResetPlayerPassword(req.PlayerID, hash)
}
