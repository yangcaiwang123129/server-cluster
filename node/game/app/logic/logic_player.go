package logic

import (
	"context"
	"errors"
	"server-cluster/internal/actor"
	"server-cluster/share/util"
	"strconv"
	"sync"
	"time"

	"github.com/dobyte/due/v2/cluster"
	"github.com/dobyte/due/v2/cluster/node"
	dueerrors "github.com/dobyte/due/v2/errors"
	"github.com/dobyte/due/v2/log"
	"github.com/dobyte/due/v2/session"
	"github.com/dobyte/due/v2/utils/xtime"
	"gorm.io/gorm"

	"server-cluster/internal/cache"
	"server-cluster/internal/svc"
	"server-cluster/node/game/app/global"
	"server-cluster/share/errs"
	"server-cluster/share/model"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
)

type playerLogic struct {
	OnlinePlayerActors *util.SafeMap[int64, *actor.PlayerActor] // playerID -> 玩家actor
}

var playerLogicInst = singleton.NewContainer[global.PlayerLogic]("PlayerLogic")

// init 装配期自注册实现。
func init() {
	playerLogicInst.Register(&playerLogic{})
}

// PlayerLogic 获取玩家模块业务单例。
func PlayerLogic() global.PlayerLogic { return playerLogicInst.MustGet() }

// EnterGame 进入游戏：playerID 由 handler 层从 ctx.UID() 取得（首包 verify 已完成绑定）。
// 先加载核心数据并驱动各玩家模块预热/建档，全部成功后才创建在线 actor。
func (this *playerLogic) EnterGame(ctx context.Context, playerID int64, req *pb.EnterGameReq) (*pb.EnterGameResp, error) {
	if _, err := global.PlayerCoreCache.Get(ctx, playerID); errors.Is(err, gorm.ErrRecordNotFound) {
		// 首次进入，建档并异步落库
		global.PlayerCoreCache.Set(playerID, model.PlayerCore{
			PlayerID: playerID,
			Nickname: "player_" + strconv.FormatInt(playerID, 10),
			Level:    1,
		})
	} else if err != nil {
		return nil, err
	}

	// 各玩家数据模块预热/建档，任一失败则不进入游戏
	for _, mod := range playerModules() {
		if _, err := mod.EnterGame(ctx, playerID, req); err != nil {
			return nil, err
		}
	}

	// 创建玩家 actor（已在线则复用），在线登记由 actor 生命周期驱动。
	if err := Online().Enter(playerID); err != nil {
		return nil, err
	}

	resp := &pb.EnterGameResp{
		Result:     &pb.Result{Code: pb.ErrCode_ERR_CODE_OK},
		ServerTime: xtime.Now().UnixMilli(),
	}
	// 当前出战英雄 / 装备武器：仅下发状态，静态属性由客户端关联配置表
	if row, err := global.PlayerHeroCache.Get(ctx, playerID); err == nil && row.Heroes != nil {
		row.Heroes.Range(func(_ int32, h model.HeroInfo) bool {
			if !h.Equipped {
				return true
			}
			resp.HeroEntry = &pb.HeroEntry{
				HeroId:   h.HeroID,
				Level:    h.Level,
				Exp:      h.Exp,
				Star:     h.Star,
				SkinId:   h.SkinID,
				Equipped: true,
			}
			return false
		})
	}
	if row, err := global.PlayerWeaponCache.Get(ctx, playerID); err == nil && row.Weapons != nil {
		row.Weapons.Range(func(_ int32, w model.WeaponInfo) bool {
			if !w.Equipped {
				return true
			}
			resp.WeaponEntry = &pb.WeaponEntry{
				WeaponId:   w.WeaponID,
				Level:      w.Level,
				Durability: w.Durability,
				Equipped:   true,
			}
			return false
		})
	}
	return resp, nil
}

// LeaveGame 离开游戏：各模块待写数据同步落库并清缓存，再销毁玩家 actor。
// 单个模块失败不阻断其余模块，返回首个错误。
func (this *playerLogic) LeaveGame(ctx context.Context, playerID int64, req *pb.LeaveGameReq) (*pb.LeaveGameResp, error) {
	var firstErr error
	record := func(name string, err error) {
		if err == nil {
			return
		}
		log.Errorf("logic: leave game flush %s failed, playerID=%d: %v", name, playerID, err)
		if firstErr == nil {
			firstErr = err
		}
	}

	// 各玩家数据模块
	for _, mod := range playerModules() {
		_, err := mod.LeaveGame(ctx, playerID, req)
		record("module", err)
	}
	// 无独立模块的玩家表：账号、核心、资料、资产
	record("player_base", flushAndEvict(ctx, global.PlayerBaseCache, playerID))
	record("player_core", flushAndEvict(ctx, global.PlayerCoreCache, playerID))
	record("player_profile", flushAndEvict(ctx, global.PlayerProfileCache, playerID))
	record("player_asset", flushAndEvict(ctx, global.PlayerAssetCache, playerID))

	Online().Leave(playerID)
	return &pb.LeaveGameResp{}, firstErr
}

// playerModules 持有玩家数据的模块，按进入/离开的统一顺序驱动。
func playerModules() []global.PlayerModuleLogic {
	return []global.PlayerModuleLogic{PlayerContainer(), Hero(), Skill(), Weapon(), Gear()}
}

// flushAndEvict 把该玩家的待写数据同步落库，并逐出内存缓存。
func flushAndEvict[V any](ctx context.Context, c *cache.Cache[int64, V], playerID int64) error {
	if err := c.FlushKey(ctx, playerID); err != nil {
		return err
	}
	c.Evict(playerID)
	return nil
}

func (this *playerLogic) PlayerGetActor(ctx context.Context, playerID int64) (*actor.PlayerActor, error) {

}
func (this *playerLogic) PlayerGetActors(ctx context.Context) (*actor.PlayerActor, error) {}
func (this *playerLogic) PlayerCreateActor(ctx context.Context, playerID int64) error     {}
func (this *playerLogic) DePlayerActor(ctx context.Context, playerID int64) error         {}
func (this *playerLogic) PlayerKick(ctx context.Context, playerID int64) error            {}

// OnlineProvider 实现 pb.PlayerServiceServer，供 admin 服等网格服务经 RPC 调用。
type OnlineProvider struct {
	pb.UnimplementedPlayerServiceServer
}

// GetOnlinePlayers 查看在线玩家列表（携带本实例 ID，供调用方按服分组）。
func (p *OnlineProvider) GetOnlinePlayers(ctx context.Context, req *pb.GetOnlinePlayersReq) (*pb.GetOnlinePlayersResp, error) {
	players := Online().List()
	nodeID := svc.NodeProxy().GetID()
	resp := &pb.GetOnlinePlayersResp{Players: make([]*pb.OnlinePlayer, 0, len(players))}
	for id, enterTime := range players {
		resp.Players = append(resp.Players, &pb.OnlinePlayer{PlayerId: id, EnterTime: enterTime, NodeId: nodeID})
	}
	return resp, nil
}

// KickPlayer 批量踢玩家下线（主动踢/封号踢），销毁 actor 并断开网关连接；失败（不在线等）玩家回列。
func (p *OnlineProvider) KickPlayer(ctx context.Context, req *pb.KickPlayerReq) (*pb.KickPlayerResp, error) {
	if len(req.PlayerIds) == 0 {
		return nil, errs.NewError(int(pb.ErrCode_ERR_CODE_INVALID_PARAMS), "invalid player ids")
	}
	resp := &pb.KickPlayerResp{}
	for _, id := range req.PlayerIds {
		if err := Online().Kick(ctx, id); err != nil {
			resp.FailedIds = append(resp.FailedIds, id)
		}
	}
	return resp, nil
}

// ── 在线登记（due actor）────────────────────────────────────────────

// actorKindPlayer 玩家 actor 类型。
const actorKindPlayer = "player"

// playerActorID 玩家 actor 唯一标识。
func playerActorID(playerID int64) string {
	return strconv.FormatInt(playerID, 10)
}

// onlineRegistry 在线玩家登记表：玩家进入游戏时创建 due actor，离开/断线/踢下线时销毁 actor。
// 在线登记由 playerProcessor 的生命周期（Start/Destroy）驱动维护。
type onlineRegistry struct {
	mu      sync.RWMutex
	players map[int64]int64 // playerID → 上线 Unix 秒
}

var onlineInst = &onlineRegistry{players: make(map[int64]int64)}

// Online 在线登记表单例。
func Online() *onlineRegistry { return onlineInst }

// Enter 玩家进入：创建玩家 actor（已存在则复用，重复进入幂等）。
func (r *onlineRegistry) Enter(playerID int64) error {
	id := playerActorID(playerID)
	if _, ok := svc.NodeProxy().Actor(actorKindPlayer, id); ok {
		return nil
	}
	_, err := svc.NodeProxy().Spawn(newPlayerProcessor,
		node.WithActorKind(actorKindPlayer),
		node.WithActorID(id),
		node.WithActorArgs(playerID))
	if errors.Is(err, dueerrors.ErrActorExists) {
		// Actor 预检与 Spawn 之间的并发竞态：他人已创建即视为进入成功。
		return nil
	}
	return err
}

// Leave 玩家离开：销毁 actor（幂等），Destroy 回调注销在线登记。
func (r *onlineRegistry) Leave(playerID int64) {
	svc.NodeProxy().Kill(actorKindPlayer, playerActorID(playerID))
}

// Kick 踢玩家下线：销毁 actor 并强制断开网关连接。
func (r *onlineRegistry) Kick(ctx context.Context, playerID int64) error {
	if !r.IsOnline(playerID) {
		return errs.NewError(int(pb.ErrCode_ERR_CODE_PLAYER_NOT_FOUND), "player is not online")
	}
	r.Leave(playerID)

	gid, err := svc.NodeProxy().LocateGate(ctx, playerID)
	if err != nil {
		return err
	}
	return svc.NodeProxy().Disconnect(ctx, &cluster.DisconnectArgs{
		GID:    gid,
		Kind:   session.User,
		Target: playerID,
		Force:  true,
	})
}

// IsOnline 玩家是否在线。
func (r *onlineRegistry) IsOnline(playerID int64) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.players[playerID]
	return ok
}

// List 全量在线快照。
func (r *onlineRegistry) List() map[int64]int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[int64]int64, len(r.players))
	for id, t := range r.players {
		out[id] = t
	}
	return out
}

// register 玩家上线登记（playerProcessor.Start 调用）。
func (r *onlineRegistry) register(playerID int64) {
	r.mu.Lock()
	r.players[playerID] = time.Now().Unix()
	r.mu.Unlock()
}

// unregister 玩家下线注销（playerProcessor.Destroy 调用）。
func (r *onlineRegistry) unregister(playerID int64) {
	r.mu.Lock()
	delete(r.players, playerID)
	r.mu.Unlock()
}

// playerProcessor 玩家 actor 处理器：生命周期与在线登记一一对应。
type playerProcessor struct {
	playerID int64
}

// newPlayerProcessor due actor Creator。
func newPlayerProcessor(actor *node.Actor, args ...any) node.Processor {
	var playerID int64
	if len(args) > 0 {
		playerID, _ = args[0].(int64)
	}
	return &playerProcessor{playerID: playerID}
}

func (p *playerProcessor) Init() {}

func (p *playerProcessor) Start() {
	onlineInst.register(p.playerID)
}

func (p *playerProcessor) Destroy() {
	onlineInst.unregister(p.playerID)
}
