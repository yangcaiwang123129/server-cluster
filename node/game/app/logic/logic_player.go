package logic

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/dobyte/due/v2/cluster"
	"github.com/dobyte/due/v2/cluster/node"
	"github.com/dobyte/due/v2/codes"
	dueerrors "github.com/dobyte/due/v2/errors"
	"github.com/dobyte/due/v2/log"
	"gorm.io/gorm"

	"server-cluster/internal/cache"
	"server-cluster/internal/svc"
	"server-cluster/node/game/app/global"
	"server-cluster/share/consts"
	"server-cluster/share/convert"
	"server-cluster/share/enum"
	"server-cluster/share/errs"
	"server-cluster/share/model"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
	"server-cluster/share/tpl"
	"server-cluster/share/util"
)

// playerCache 玩家账号与货币（player，一玩家一行）；读走 LRU，写异步批量 upsert。
var playerCache = cache.NewPlayerTableCache[model.Player](cache.CacherConfig{Name: enum.TablePlayer.String()})

// playerExtraCache 玩家核心数值（player_extra，一玩家一行）。
var playerExtraCache = cache.NewPlayerTableCache[model.PlayerExtra](cache.CacherConfig{Name: enum.TablePlayerExtra.String()})

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
func moduleLogics() []global.PlayerLifecycle {
	return []global.PlayerLifecycle{Item(), Hero(), Weapon(), Skill(), Equip(), Mail(), Task()}
}

// PlayerLogic 获取玩家模块业务单例。
func PlayerLogic() global.PlayerLogic { return playerLogicInst.MustGet() }

// Load 读玩家核心数据（player_extra，一玩家一行）；未建档则初始化默认昵称/等级并挂入缓存（异步落库）。
// 语义同 Cache.GetOrInit。
func (this *playerLogic) Load(ctx context.Context, playerID int64) (*model.PlayerExtra, error) {
	return playerExtraCache.GetOrInit(ctx, playerID,
		func(id int64) *model.PlayerExtra {
			return &model.PlayerExtra{
				PlayerID:     id,
				Nickname:     consts.DefaultNicknamePrefix + strconv.FormatInt(id, 10),
				Level:        consts.DefaultPlayerLevel,
				SettingsJSON: consts.DefaultPlayerSettingsJSON,
			}
		}, nil)
}

// Save 写回玩家核心数据并挂入异步落库队列。
func (this *playerLogic) Save(playerID int64, row *model.PlayerExtra) {
	playerExtraCache.Set(playerID, row)
}

// Flush 同步落库并逐出玩家核心数据缓存（GM 直改 DB 后强制回源）。
func (this *playerLogic) Flush(ctx context.Context, playerID int64) error {
	return flushAndEvict(ctx, playerExtraCache, playerID)
}

// OnEnterGame 进入游戏：playerID 由 handler 层从 ctx.UID() 取得（首包 verify 已完成绑定与 actor 创建）。
// 加载核心数据并驱动各玩家模块预热/建档。
func (this *playerLogic) OnEnterGame(ctx context.Context, playerID int64, req *pb.EnterGameReq, resp *pb.EnterGameResp) error {
	core, err := this.Load(ctx, playerID)
	if err != nil {
		return err
	}

	// 账号行由 login 服建档；异常缺失时按全 0 货币处理，不阻断进入游戏
	player, err := playerCache.Get(ctx, playerID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		player = &model.Player{PlayerID: playerID}
	} else if err != nil {
		return err
	}

	// 玩家基础属性写入响应，再由各模块把各自的初始状态写入同一响应；任一失败则不进入游戏
	resp.PlayerAttr = convert.PlayerAttr(core, player)
	for _, mod := range moduleLogics() {
		if err := mod.OnEnterGame(ctx, playerID, req, resp); err != nil {
			return err
		}
	}

	resp.Activities = ActivitySync().Activities()
	resp.Result = errs.BuildOKResult()
	resp.ServerTime = util.Now().UnixMilli()
	return nil
}

// 玩家属性表组装：见 convert.PlayerAttr。

// LeaveGame 离开游戏：各模块待写数据同步落库并清缓存，再销毁玩家 actor。
// 单个模块失败不阻断其余模块，返回首个错误。
func (this *playerLogic) OnLeaveGame(ctx context.Context, playerID int64, req *pb.LeaveGameReq, resp *pb.LeaveGameResp) error {
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
		record("module", mod.OnLeaveGame(ctx, playerID, req, resp))
	}
	// 无独立模块的玩家表：账号（含货币）
	record("player", flushAndEvict(ctx, playerCache, playerID))
	// 核心数据（player_extra）走模块 Flush
	record("player_core", this.Flush(ctx, playerID))

	record("player_actor", this.DestroyActor(ctx, playerID))
	return firstErr
}

// ── 玩家资产消费：礼包购买 / 兑换 / 充值 ────────────────────────────

// GiftBuy 购买礼包：按 GiftCfg 扣除 AssetCost 并发放 AssetRewards。
// 配置不存在或资产不足返回错误，已扣减的部分不回滚（与现有资产逻辑一致）。
func (this *playerLogic) GiftBuy(ctx context.Context, playerID int64, req *pb.GiftBuyReq) (*pb.GiftBuyResp, error) {
	cfg := tpl.T().Gift.Get(req.GetCfgId())
	if cfg == nil {
		return &pb.GiftBuyResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "礼包配置不存在")}, nil
	}

	costs, err := deductAssets(ctx, playerID, []*tpl.Asset{cfg.AssetCost})
	if err != nil {
		if r := codedResult(err); r != nil {
			return &pb.GiftBuyResp{Result: r}, nil
		}
		return nil, err
	}
	rewards, err := grantAssets(ctx, playerID, cfg.AssetRewards)
	if err != nil {
		if r := codedResult(err); r != nil {
			return &pb.GiftBuyResp{Result: r}, nil
		}
		return nil, err
	}
	return &pb.GiftBuyResp{Costs: costs, Rewards: rewards}, nil
}

// Exchange 兑换：按 ExchangeCfg 扣除 AssetCost 并发放 AssetRewards。
// 配置不存在或资产不足返回错误，已扣减的部分不回滚（与现有资产逻辑一致）。
func (this *playerLogic) Exchange(ctx context.Context, playerID int64, req *pb.ExchangeReq) (*pb.ExchangeResp, error) {
	cfg := tpl.T().Exchange.Get(req.GetCfgId())
	if cfg == nil {
		return &pb.ExchangeResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "兑换配置不存在")}, nil
	}

	costs, err := deductAssets(ctx, playerID, []*tpl.Asset{cfg.AssetCost})
	if err != nil {
		if r := codedResult(err); r != nil {
			return &pb.ExchangeResp{Result: r}, nil
		}
		return nil, err
	}
	rewards, err := grantAssets(ctx, playerID, cfg.AssetRewards)
	if err != nil {
		if r := codedResult(err); r != nil {
			return &pb.ExchangeResp{Result: r}, nil
		}
		return nil, err
	}
	return &pb.ExchangeResp{Costs: costs, Rewards: rewards}, nil
}

// notifyRechargeOrder 预留：通知后台充值订单。
// 接入支付 SDK 后，这里改为携带订单号回调支付平台/后台校验，校验通过再发放奖励；
// 当前是假充值，只记录一条日志，便于后续替换。
var notifyRechargeOrder = func(playerID int64, cfgID int32) {
	log.Infof("logic: recharge order notify reserved, playerID=%d, cfgID=%d", playerID, cfgID)
}

// Recharge 充值（假充值）：校验配置后直接发放 AssetReward，不做订单校验。
func (this *playerLogic) Recharge(ctx context.Context, playerID int64, req *pb.RechargeReq) (*pb.RechargeResp, error) {
	cfg := tpl.T().Recharge.Get(req.GetCfgId())
	if cfg == nil {
		return &pb.RechargeResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "充值配置不存在")}, nil
	}

	notifyRechargeOrder(playerID, req.GetCfgId())
	rewards, err := grantAssets(ctx, playerID, []*tpl.Asset{cfg.AssetReward})
	if err != nil {
		if r := codedResult(err); r != nil {
			return &pb.RechargeResp{Result: r}, nil
		}
		return nil, err
	}
	// 累计充值次数（player_extra），加载失败不阻断充值流程
	if extra, err := this.Load(ctx, playerID); err == nil && extra != nil {
		extra.TotalRecharge++
		this.Save(playerID, extra)
	}
	// 固定充值（按配置ID匹配）与累计充值（全匹配）双事件
	Task().EmitTaskEvent(ctx, global.TaskEvent{
		PlayerID:       playerID,
		TargetTaskType: int32(tpl.TargetTaskType_FixedRecharge),
		Param:          req.GetCfgId(),
		Count:          1,
	})
	Task().EmitTaskEvent(ctx, global.TaskEvent{
		PlayerID:       playerID,
		TargetTaskType: int32(tpl.TargetTaskType_CumulativeRecharge),
		Count:          1,
	})
	return &pb.RechargeResp{Rewards: rewards}, nil
}

// flushAndEvict 把该玩家的待写数据同步落库，并逐出内存缓存。
func flushAndEvict[V any](ctx context.Context, c *cache.Cache[int64, V], playerID int64) error {
	if err := c.FlushKey(ctx, playerID); err != nil {
		return err
	}
	c.Evict(playerID)
	return nil
}

// codedResult helper 链路上的业务码错误转 Result（仅 logic 层内部使用）。
// 内部 helper 保留 errs.CodeXxx 错误返回，logic 方法在此统一转成客户端 Result；
// 普通基础设施错误（codes.Convert 解析为 Unknown）返回 nil，由调用方原样上抛，
// 最终由 handler 兜底为 INTERNAL_ERROR。
func codedResult(err error) *pb.Result {
	c := codes.Convert(err)
	if c == nil || c.Code() == codes.OK.Code() || c.Code() == codes.Unknown.Code() {
		return nil
	}
	return errs.BuildErrResult(pb.ErrCode(c.Code()), c.Message())
}

// GetActor 获取玩家 actor；玩家不在线时返回玩家不存在。
func (this *playerLogic) GetActor(ctx context.Context, playerID int64) (*global.PlayerActor, error) {
	if pa, ok := this.PlayerActors.Get(playerID); ok {
		return pa, nil
	}
	return nil, errs.ErrOf(pb.ErrCode_ERR_CODE_PLAYER_NOT_FOUND, "player actor not found")
}

// GetActors 获取全部在线玩家 actor（快照，可在遍历外安全使用）。
func (this *playerLogic) GetActors(ctx context.Context) ([]*global.PlayerActor, error) {
	actors := make([]*global.PlayerActor, 0, this.PlayerActors.Len())
	this.PlayerActors.Range(func(_ int64, pa *global.PlayerActor) bool {
		actors = append(actors, pa)
		return true
	})
	return actors, nil
}

// CreateActor 创建玩家 actor（已在线则幂等复用）。
// loginIP 从 node.Context.GetIP() 取，存入内存，不落库。
// 在线登记由 playerProcessor 的 Start/Destroy 生命周期维护。
func (this *playerLogic) CreateActor(ctx context.Context, playerID int64, loginIP string) error {
	if this.PlayerActors.Has(playerID) {
		// 已在线：幂等更新登录 IP 与重连时间（断线重连可能换 IP；ReconnectAt 供断线延迟销毁豁免判断）
		if pa, ok := this.PlayerActors.Get(playerID); ok {
			pa.LoginIP = loginIP
			pa.ReconnectAt = util.Now().Unix()
		}
		return nil
	}

	_, err := svc.NodeProxy().Spawn(newPlayerProcessor,
		node.WithActorKind(actorKindPlayer),
		node.WithActorID(playerActorID(playerID)),
		node.WithActorArgs(playerID, loginIP))
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
		EnterTime: util.Now().Unix(),
		LoginIP:   loginIP,
	})
	return nil
}

// DestroyActor 销毁玩家 actor（幂等），Destroy 回调同步注销在线登记。
// 同时解绑 redis locate（gate/node 路由），防止已删/已下线玩家的路由残留导致
// locate 表只进不出、网关日志与后台玩家数据对不上。
func (this *playerLogic) DestroyActor(ctx context.Context, playerID int64) error {
	svc.NodeProxy().Kill(actorKindPlayer, playerActorID(playerID))
	this.PlayerActors.Delete(playerID)

	if nc, ok := ctx.(node.Context); ok {
		_ = nc.UnbindGate(playerID)
		_ = nc.UnbindNode(playerID)
	}
	return nil
}

// DestroyActorIfOffline 断线延迟销毁：仅当玩家在 since（断开时刻）之后未重连时才销毁 actor。
// 同 uid 重连/互踢场景下旧连接的 Disconnect 不能删掉新连接正在使用的 actor。
func (this *playerLogic) DestroyActorIfOffline(ctx context.Context, playerID int64, since int64) error {
	pa, ok := this.PlayerActors.Get(playerID)
	if !ok {
		return nil // actor 已不存在（正常走 LeaveGame 清理），幂等
	}
	if pa.ReconnectAt >= since {
		return nil // 断开后已重连（EnterGame 复用 actor 更新了 ReconnectAt），保留
	}
	return this.DestroyActor(ctx, playerID)
}

// Kick 踢玩家下线：销毁 actor（注销在线登记）并强制断开网关连接。
func (this *playerLogic) Kick(ctx context.Context, playerID int64) error {
	if !this.PlayerActors.Has(playerID) {
		return errs.ErrOf(pb.ErrCode_ERR_CODE_PLAYER_NOT_FOUND, "player is not online")
	}
	if err := this.DestroyActor(ctx, playerID); err != nil {
		return err
	}

	gid, err := svc.NodeProxy().LocateGate(ctx, playerID)
	if err != nil {
		return err
	}
	return svc.NodeProxy().Disconnect(ctx, &cluster.DisconnectArgs{
		GID:    gid,
		Kind:   cluster.User,
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
	loginIP  string
}

// newPlayerProcessor due actor Creator。
func newPlayerProcessor(act *node.Actor, args ...any) node.Processor {
	var playerID int64
	var loginIP string
	if len(args) > 0 {
		playerID, _ = args[0].(int64)
	}
	if len(args) > 1 {
		loginIP, _ = args[1].(string)
	}
	return &playerProcessor{actor: act, playerID: playerID, loginIP: loginIP}
}

// Start 玩家 actor 启动：写在线登记、绑定 uid 与 actor，并把业务路由注册到 actor。
func (p *playerProcessor) Start() {
	playerActorMap().Set(p.playerID, &global.PlayerActor{
		Actor:     p.actor,
		PlayerID:  p.playerID,
		EnterTime: util.Now().Unix(),
		LoginIP:   p.loginIP,
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

// CacheCloseStat 单个玩家数据缓存在停服刷盘时的统计，用于停服日志展示。
type CacheCloseStat struct {
	Name string
	Rows int
	Cost time.Duration
}

// cacheCloser 停服日志所需的缓存能力（泛型缓存实例的公共方法集合）。
type cacheCloser interface {
	Name() string
	PendingLen() int
	Close()
}

// CloseCaches 停止后台落库协程，把剩余待写数据刷入 MySQL，返回每个缓存的刷盘统计。
func CloseCaches() []CacheCloseStat {
	closeOne := func(c cacheCloser) CacheCloseStat {
		begin := time.Now()
		rows := c.PendingLen()
		c.Close()
		return CacheCloseStat{Name: c.Name(), Rows: rows, Cost: time.Since(begin)}
	}
	return []CacheCloseStat{
		closeOne(playerCache),
		closeOne(playerExtraCache),
		closeOne(heroCache),
		closeOne(weaponCache),
		closeOne(skillCache),
		closeOne(equipCache),
		closeOne(itemCache),
		closeOne(mailCache),
		closeOne(taskCache),
		closeOne(activityCache),
	}
}
