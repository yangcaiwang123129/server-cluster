package logic

import (
	"context"
	"errors"
	"strconv"

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
	"server-cluster/share/consts"
	"server-cluster/share/errs"
	"server-cluster/share/model"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
	"server-cluster/share/util"
)

// actorKindPlayer 玩家 actor 类型。
const actorKindPlayer = "player"

type playerLogic struct {
	PlayerActors *util.SafeMap[int64, *global.PlayerActor] // playerID -> 玩家actor
}

var playerLogicInst = singleton.NewContainer[global.PlayerLogic]("PlayerLogic")

func init() {
	playerLogicInst.Register(&playerLogic{PlayerActors: util.NewSafeMap[int64, *global.PlayerActor]()})
}

// moduleLogics 持有玩家数据的模块，按进入/离开的统一顺序驱动。
// 延迟获取单例：各模块的 init 注册顺序不确定，不能在包初始化阶段调用。
func moduleLogics() []global.ModuleLogic {
	return []global.ModuleLogic{Item(), Hero(), Skill(), Weapon(), Equip()}
}

// PlayerLogic 获取玩家模块业务单例。
func PlayerLogic() global.PlayerLogic { return playerLogicInst.MustGet() }

// EnterGame 进入游戏：playerID 由 handler 层从 ctx.UID() 取得（首包 verify 已完成绑定）。
// 先加载核心数据并驱动各玩家模块预热/建档，全部成功后才创建在线 actor。
func (this *playerLogic) EnterGame(ctx context.Context, playerID int64, req *pb.EnterGameReq) (*pb.EnterGameResp, error) {
	core, err := global.PlayerExtraCache.Get(ctx, playerID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 首次进入，建档并异步落库
		core = &model.PlayerExtra{
			PlayerID:     playerID,
			Nickname:     consts.DefaultNicknamePrefix + strconv.FormatInt(playerID, 10),
			Level:        consts.DefaultPlayerLevel,
			SettingsJSON: consts.DefaultPlayerSettingsJSON,
		}
		global.PlayerExtraCache.Set(playerID, core)
	} else if err != nil {
		return nil, err
	}

	// 账号行由 login 服建档；异常缺失时按全 0 货币处理，不阻断进入游戏
	player, err := global.PlayerCache.Get(ctx, playerID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		player = &model.Player{PlayerID: playerID}
	} else if err != nil {
		return nil, err
	}

	// 各玩家数据模块预热/建档，并把各自需要下发的初始状态合并进响应；任一失败则不进入游戏
	resp := &pb.EnterGameResp{PlayerMagicAttr: buildPlayerMagicAttr(core, player)}
	for _, mod := range moduleLogics() {
		part, err := mod.EnterGame(ctx, playerID, req)
		if err != nil {
			return nil, err
		}
		mergeEnterGameResp(resp, part)
	}

	// 创建玩家 actor（已在线则复用），在线登记由 actor 生命周期驱动。
	if err := this.PlayerCreateActor(ctx, playerID); err != nil {
		return nil, err
	}

	resp.Result = &pb.Result{Code: pb.ErrCode_ERR_CODE_OK}
	resp.ServerTime = xtime.Now().UnixMilli()
	return resp, nil
}

// mergeEnterGameResp 合并模块返回的初始状态（只取模块实际填充的字段）。
func mergeEnterGameResp(dst, src *pb.EnterGameResp) {
	if src == nil {
		return
	}
	if src.HeroEntry != nil {
		dst.HeroEntry = src.HeroEntry
	}
	if src.WeaponEntry != nil {
		dst.WeaponEntry = src.WeaponEntry
	}
}

// buildPlayerMagicAttr 组装下发给客户端的玩家属性表：
// 昵称/等级/经验/vip 取自 player_extra，币/金取自 player。
// 键为 MagicAttr 枚举值，值按实际类型放入 ObjectValue。
func buildPlayerMagicAttr(core *model.PlayerExtra, player *model.Player) *pb.MagicAttrMap {
	return &pb.MagicAttrMap{Values: map[int32]*pb.ObjectValue{
		int32(pb.MagicAttr_PLAYER_ATTR_NAME):      {Kind: &pb.ObjectValue_StringValue{StringValue: core.Nickname}},
		int32(pb.MagicAttr_PLAYER_ATTR_LEVEL):     {Kind: &pb.ObjectValue_IntValue{IntValue: core.Level}},
		int32(pb.MagicAttr_PLAYER_ATTR_EXP):       {Kind: &pb.ObjectValue_BigintValue{BigintValue: core.Exp}},
		int32(pb.MagicAttr_PLAYER_ATTR_VIP_LEVEL): {Kind: &pb.ObjectValue_IntValue{IntValue: core.VipLevel}},
		int32(pb.MagicAttr_PLAYER_ATTR_VIP_EXP):   {Kind: &pb.ObjectValue_BigintValue{BigintValue: core.VipExp}},
		int32(pb.MagicAttr_PLAYER_ATTR_COIN):      {Kind: &pb.ObjectValue_BigintValue{BigintValue: player.Coin}},
		int32(pb.MagicAttr_PLAYER_ATTR_GOLD):      {Kind: &pb.ObjectValue_BigintValue{BigintValue: player.Gold}},
	}}
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
	for _, mod := range moduleLogics() {
		_, err := mod.LeaveGame(ctx, playerID, req)
		record("module", err)
	}
	// 无独立模块的玩家表：账号（含货币）、核心
	record("player", flushAndEvict(ctx, global.PlayerCache, playerID))
	record("player_core", flushAndEvict(ctx, global.PlayerExtraCache, playerID))

	record("player_actor", this.PlayerDestroyActor(ctx, playerID))
	return &pb.LeaveGameResp{}, firstErr
}

// flushAndEvict 把该玩家的待写数据同步落库，并逐出内存缓存。
func flushAndEvict[V any](ctx context.Context, c *cache.Cache[int64, V], playerID int64) error {
	if err := c.FlushKey(ctx, playerID); err != nil {
		return err
	}
	c.Evict(playerID)
	return nil
}

// PlayerGetActor 获取玩家 actor；玩家不在线时返回玩家不存在。
func (this *playerLogic) PlayerGetActor(ctx context.Context, playerID int64) (*global.PlayerActor, error) {
	if pa, ok := this.PlayerActors.Get(playerID); ok {
		return pa, nil
	}
	return nil, errs.NewError(int(pb.ErrCode_ERR_CODE_PLAYER_NOT_FOUND), "player actor not found")
}

// PlayerGetActors 获取全部在线玩家 actor（快照，可在遍历外安全使用）。
func (this *playerLogic) PlayerGetActors(ctx context.Context) ([]*global.PlayerActor, error) {
	actors := make([]*global.PlayerActor, 0, this.PlayerActors.Len())
	this.PlayerActors.Range(func(_ int64, pa *global.PlayerActor) bool {
		actors = append(actors, pa)
		return true
	})
	return actors, nil
}

// PlayerCreateActor 创建玩家 actor（已在线则幂等复用）。
// 在线登记由 playerProcessor 的 Start/Destroy 生命周期维护。
func (this *playerLogic) PlayerCreateActor(ctx context.Context, playerID int64) error {
	if this.PlayerActors.Has(playerID) {
		return nil
	}

	_, err := svc.NodeProxy().Spawn(newPlayerProcessor,
		node.WithActorKind(actorKindPlayer),
		node.WithActorID(playerActorID(playerID)),
		node.WithActorArgs(playerID))
	if err == nil {
		return nil
	}
	if !errors.Is(err, dueerrors.ErrActorExists) {
		return err
	}

	// 并发重复创建：actor 已由他人创建，读回并补齐本地登记。
	pa, ok := svc.NodeProxy().Actor(actorKindPlayer, playerActorID(playerID))
	if !ok {
		return err
	}
	this.PlayerActors.Set(playerID, &global.PlayerActor{
		Actor:     pa,
		PlayerID:  playerID,
		EnterTime: xtime.Now().Unix(),
	})
	return nil
}

// PlayerDestroyActor 销毁玩家 actor（幂等），Destroy 回调同步注销在线登记。
func (this *playerLogic) PlayerDestroyActor(ctx context.Context, playerID int64) error {
	svc.NodeProxy().Kill(actorKindPlayer, playerActorID(playerID))
	this.PlayerActors.Delete(playerID)
	return nil
}

// PlayerKick 踢玩家下线：销毁 actor（注销在线登记）并强制断开网关连接。
func (this *playerLogic) PlayerKick(ctx context.Context, playerID int64) error {
	if !this.PlayerActors.Has(playerID) {
		return errs.NewError(int(pb.ErrCode_ERR_CODE_PLAYER_NOT_FOUND), "player is not online")
	}
	if err := this.PlayerDestroyActor(ctx, playerID); err != nil {
		return err
	}

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

// ── 玩家 actor（due actor）────────────────────────────────────────────

// playerActorID 玩家 actor 唯一标识。
func playerActorID(playerID int64) string {
	return strconv.FormatInt(playerID, 10)
}

// playerActorMap 在线玩家 actor 表（与 PlayerLogic 单例共享同一实例）。
func playerActorMap() *util.SafeMap[int64, *global.PlayerActor] {
	return playerLogicInst.MustGet().(*playerLogic).PlayerActors
}

// actorRouteRegistrar 玩家 actor 路由注册器，由 handler 包装配期注入（避免 logic → handler 循环依赖）。
var actorRouteRegistrar func(act *node.Actor)

// RegisterActorRouteRegistrar 注入玩家 actor 路由注册器（handler 包在装配期调用）。
func RegisterActorRouteRegistrar(fn func(act *node.Actor)) {
	actorRouteRegistrar = fn
}

// playerProcessor 玩家 actor 处理器：生命周期与在线登记一一对应。
type playerProcessor struct {
	node.BaseProcessor
	actor    *node.Actor
	playerID int64
}

// newPlayerProcessor due actor Creator。
func newPlayerProcessor(act *node.Actor, args ...any) node.Processor {
	var playerID int64
	if len(args) > 0 {
		playerID, _ = args[0].(int64)
	}
	return &playerProcessor{actor: act, playerID: playerID}
}

// Start 玩家 actor 启动：写在线登记、绑定 uid 与 actor，并把业务路由注册到 actor。
func (p *playerProcessor) Start() {
	playerActorMap().Set(p.playerID, &global.PlayerActor{
		Actor:     p.actor,
		PlayerID:  p.playerID,
		EnterTime: xtime.Now().Unix(),
	})

	// uid 与 actor 绑定：stateful 消息由 scheduler 定位到本 actor
	if err := p.actor.Proxy().BindActor(p.playerID, actorKindPlayer, p.actor.ID()); err != nil {
		log.Errorf("logic: bind player actor failed, playerID=%d: %v", p.playerID, err)
	}
	// 注册业务路由：消息进入 actor 后串行处理
	if actorRouteRegistrar != nil {
		actorRouteRegistrar(p.actor)
	}
}

// Destroy 玩家 actor 销毁：注销在线登记。
func (p *playerProcessor) Destroy() {
	playerActorMap().Delete(p.playerID)
}
