package logic

import (
	"context"
	"crypto/md5"
	"encoding/hex"

	"server-cluster/internal/rpc"
	"server-cluster/mesh/admin/app/global"
	"server-cluster/mesh/admin/app/repo"
	"server-cluster/share/dto"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
	"server-cluster/share/util"

	"github.com/dobyte/due/v2/codes"
)

// playerLogicInst 玩家业务单例容器。
var playerLogicInst = singleton.NewContainer[global.PlayerLogic]("PlayerLogic")

// RegisterPlayerLogic 装配期注册玩家业务实现。
func RegisterPlayerLogic(inst global.PlayerLogic) { playerLogicInst.Register(inst) }

// Player 获取玩家模块业务单例。
func Player() global.PlayerLogic { return playerLogicInst.MustGet() }

// playerLogic 玩家管理业务实现。
type playerLogic struct{}

func init() {
	RegisterPlayerLogic(&playerLogic{})
}

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

func (this *playerLogic) PlayerOnline() ([]dto.OnlinePlayerResp, error) {
	instances, err := gameInstances(context.Background(), pb.GameAdminService_ServiceDesc.ServiceName)
	if err != nil {
		return nil, err
	}

	list := make([]dto.OnlinePlayerResp, 0)
	for _, r := range rpc.CallAll[pb.GetOnlinePlayersResp](context.Background(), instanceIDs(instances),
		pb.GameAdminService_GetOnlinePlayers_FullMethodName, &pb.GetOnlinePlayersReq{}) {
		if r.Err != nil {
			continue // 实例不可达：跳过，避免单实例故障拖垮整个列表
		}
		for _, p := range r.Resp.Players {
			list = append(list, dto.OnlinePlayerResp{
				PlayerID:    p.PlayerId,
				EnterTime:   p.EnterTime,
				LobbyNodeID: p.NodeId,
				LoginIP:     p.LoginIp,
			})
		}
	}
	return list, nil
}

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
	rpc.CallAll[pb.KickPlayerResp](context.Background(), instanceIDs(instances),
		pb.GameAdminService_KickPlayer_FullMethodName, &pb.KickPlayerReq{PlayerIds: []int64{req.PlayerID}})
	return nil
}

func (this *playerLogic) PlayerUnban(req dto.PlayerUnbanReq) error {
	return repo.R().UnbanPlayer(req.PlayerID)
}

func (this *playerLogic) PlayerResetPassword(req dto.PlayerResetPasswordReq) error {
	h := md5.Sum([]byte(req.NewPassword))
	hash := hex.EncodeToString(h[:])
	return repo.R().ResetPlayerPassword(req.PlayerID, hash)
}
