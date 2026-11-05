package logic

import (
	"context"
	"encoding/json"
	"errors"
	"server-cluster/internal/actor"
	"server-cluster/node/game/app/repo"
	"strconv"
	"strings"
	"unicode/utf8"

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

// actorKindPlayer 玩家 actor 类型。
const actorKindPlayer = "player"

type playerLogic struct {
	// cache 玩家核心数值（player_extra，一玩家一行）。
	cache *cache.Cache[int64, *model.PlayerExtra]
	// attrHandlers 属性类型 → 操作实现（见 newAttrHandlers）。
	attrHandlers map[pb.Attr]attrHandler
	// PlayerActors 在线玩家 actor 表（playerID -> 玩家 actor）。
	PlayerActors *util.SafeMap[int64, *actor.PlayerActor]
}

var playerLogicInst = singleton.NewContainer[global.PlayerLogic]("PlayerLogic")

func init() {
	pLogic := &playerLogic{
		cache:        cache.NewPlayerTableCache[model.PlayerExtra](cache.CacherConfig{Name: enum.TablePlayerExtra.String()}),
		attrHandlers: newAttrHandlers(),
		PlayerActors: util.NewSafeMap[int64, *actor.PlayerActor](),
	}

	playerLogicInst.Register(pLogic)
	cache.RegisterCacheCloser(pLogic.cache)
}

// moduleLogics 延迟获取持有玩家数据的模块（init 注册顺序不确定，不能在包初始化阶段调用）。
func moduleLogics() []global.PlayerLifecycle {
	return []global.PlayerLifecycle{Item(), Hero(), Weapon(), Skill(), Equip(), Mail(), Task()}
}

// PlayerLogic 获取玩家模块业务单例。
func PlayerLogic() global.PlayerLogic { return playerLogicInst.MustGet() }

func (this *playerLogic) Load(ctx context.Context, playerID int64) (*model.PlayerExtra, error) {
	return this.cache.GetOrInit(ctx, playerID,
		func(id int64) *model.PlayerExtra {
			return &model.PlayerExtra{
				PlayerID:     id,
				Nickname:     consts.DefaultNicknamePrefix + strconv.FormatInt(id, 10),
				Level:        consts.DefaultPlayerLevel,
				SettingsJSON: consts.DefaultPlayerSettingsJSON,
			}
		}, nil)
}

func (this *playerLogic) Save(playerID int64, row *model.PlayerExtra) {
	this.cache.Set(playerID, row)
}

func (this *playerLogic) Flush(ctx context.Context, playerID int64) error {
	return flushAndEvict(ctx, this.cache, playerID)
}

func (this *playerLogic) OnEnterGame(ctx context.Context, playerID int64, req *pb.EnterGameReq, resp *pb.EnterGameResp) error {
	core, err := this.Load(ctx, playerID)
	if err != nil {
		return err
	}

	// 账号行由 login 服建档，game 侧直查 DB（不缓存）；异常缺失时按全 0 货币处理，不阻断进入游戏
	player, err := repo.QueryPlayer(ctx, playerID)
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
	// 核心数据（player_extra）走模块 Flush
	record("player_extra", this.Flush(ctx, playerID))
	record("player_actor", this.DestroyActor(ctx, playerID))
	return firstErr
}

// ── 玩家资产消费：礼包购买 / 兑换 / 充值 ────────────────────────────

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

// codedResult 业务码错误转客户端 Result；基础设施错误（Unknown）返回 nil 由调用方上抛，handler 兜底 INTERNAL_ERROR。
func codedResult(err error) *pb.Result {
	c := codes.Convert(err)
	if c == nil || c.Code() == codes.OK.Code() || c.Code() == codes.Unknown.Code() {
		return nil
	}
	return errs.BuildErrResult(pb.ErrCode(c.Code()), c.Message())
}

// ── 玩家 actor 基础设施：唯一标识、在线登记表、处理器与路由注册器注入 ──
// actor 生命周期与在线登记一一对应：processor.Start 时写入 PlayerActors 并绑定 uid、
// 注册业务路由，processor.Destroy 时从 PlayerActors 移除登记。

// playerActorID 玩家 actor 唯一标识。
func playerActorID(playerID int64) string {
	return strconv.FormatInt(playerID, 10)
}

// playerActorMap 获取在线玩家 actor 表。
func playerActorMap() *util.SafeMap[int64, *actor.PlayerActor] {
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

func (p *playerProcessor) Start() {
	playerActorMap().Set(p.playerID, &actor.PlayerActor{
		Actor:     p.actor,
		PlayerID:  p.playerID,
		EnterTime: util.Now().Unix(),
		LoginIP:   p.loginIP,
	})

	// uid 与 actor 绑定：stateful 消息由 scheduler 定位到本 actor
	if err := p.actor.Proxy().BindActor(p.playerID, actorKindPlayer, p.actor.ID()); err != nil {
		log.Errorf("game: bind player actor failed, playerID=%d: %v", p.playerID, err)
	}
	// 注册业务路由：消息进入 actor 后串行处理
	if actorRouteRegistrar != nil {
		actorRouteRegistrar(p.actor)
	}
}

func (p *playerProcessor) Destroy() {
	playerActorMap().Delete(p.playerID)
}

func (this *playerLogic) GetActor(ctx context.Context, playerID int64) (*actor.PlayerActor, error) {
	if pa, ok := this.PlayerActors.Get(playerID); ok {
		return pa, nil
	}
	return nil, errs.ErrOf(pb.ErrCode_ERR_CODE_PLAYER_NOT_FOUND, "player actor not found")
}

func (this *playerLogic) GetActors(ctx context.Context) ([]*actor.PlayerActor, error) {
	actors := make([]*actor.PlayerActor, 0, this.PlayerActors.Len())
	this.PlayerActors.Range(func(_ int64, pa *actor.PlayerActor) bool {
		actors = append(actors, pa)
		return true
	})
	return actors, nil
}

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
	this.PlayerActors.Set(playerID, &actor.PlayerActor{
		Actor:     pa,
		PlayerID:  playerID,
		EnterTime: util.Now().Unix(),
		LoginIP:   loginIP,
	})
	return nil
}

func (this *playerLogic) DestroyActor(ctx context.Context, playerID int64) error {
	svc.NodeProxy().Kill(actorKindPlayer, playerActorID(playerID))
	this.PlayerActors.Delete(playerID)

	if nc, ok := ctx.(node.Context); ok {
		_ = nc.UnbindGate(playerID)
		_ = nc.UnbindNode(playerID)
	}
	return nil
}

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

// attrHandler 单个属性类型的操作实现。
type attrHandler struct {
	// 属性更新实现：校验并写入 core，返回操作后的属性值。
	update func(core *model.PlayerExtra, value *pb.ObjectValue) (*pb.ObjectValue, error)
	// 属性领取实现：按玩家状态发放奖励，返回操作后的属性值。
	claim func(ctx context.Context, playerID int64, core *model.PlayerExtra) (*pb.ObjectValue, error)
}

// newAttrHandlers 属性操作注册表；未登记的类型或未实现的操作一律视为不支持。
// 1000~4999 任务/成就进度、5000~9999 活动状态等属性接入时在此扩展。
func newAttrHandlers() map[pb.Attr]attrHandler {
	return map[pb.Attr]attrHandler{
		pb.Attr_PLAYER_ATTR_NAME:          {update: updateNickname},
		pb.Attr_PLAYER_ATTR_SIGNATURE:     {update: updateSignature},
		pb.Attr_PLAYER_ATTR_SETTINGS_JSON: {update: updateSettingsJSON},
	}
}

func (this *playerLogic) AttrOperate(ctx context.Context, playerID int64, req *pb.AttrOperateReq) (*pb.AttrOperateResp, error) {
	attr := pb.Attr(req.GetAttr())
	h, ok := this.attrHandlers[attr]
	if !ok {
		return &pb.AttrOperateResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "不支持的属性类型")}, nil
	}

	playerExtra, err := this.Load(ctx, playerID)
	if err != nil {
		return nil, err
	}

	var out *pb.ObjectValue
	switch req.GetOperate() {
	case pb.AttrOperate_UPDATE:
		if h.update == nil {
			return &pb.AttrOperateResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "该属性不支持更新")}, nil
		}
		out, err = h.update(playerExtra, req.GetValue())
	case pb.AttrOperate_CLAIM:
		if h.claim == nil {
			return &pb.AttrOperateResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "该属性不支持领取")}, nil
		}
		out, err = h.claim(ctx, playerID, playerExtra)
	default:
		return &pb.AttrOperateResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "不支持的操作类型")}, nil
	}
	if err != nil {
		if r := codedResult(err); r != nil {
			return &pb.AttrOperateResp{Result: r}, nil
		}
		return nil, err
	}

	this.Save(playerID, playerExtra)
	return &pb.AttrOperateResp{Values: map[int32]*pb.ObjectValue{int32(attr): out}}, nil
}

// updateNickname 更新昵称：去首尾空白，长度 1~MaxNicknameLen 字符。
func updateNickname(core *model.PlayerExtra, value *pb.ObjectValue) (*pb.ObjectValue, error) {
	name := strings.TrimSpace(value.GetStringValue())
	n := utf8.RuneCountInString(name)
	if n == 0 || n > consts.MaxNicknameLen {
		return nil, errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "昵称长度不合法")
	}
	core.Nickname = name
	return &pb.ObjectValue{Kind: &pb.ObjectValue_StringValue{StringValue: core.Nickname}}, nil
}

// updateSignature 更新个性签名：长度不超过 MaxSignatureLen 字符，允许为空。
func updateSignature(core *model.PlayerExtra, value *pb.ObjectValue) (*pb.ObjectValue, error) {
	signature := strings.TrimSpace(value.GetStringValue())
	if utf8.RuneCountInString(signature) > consts.MaxSignatureLen {
		return nil, errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "个性签名过长")
	}
	core.Signature = signature
	return &pb.ObjectValue{Kind: &pb.ObjectValue_StringValue{StringValue: core.Signature}}, nil
}

// updateSettingsJSON 更新个人设置：必须是 JSON 对象且不超过 MaxSettingsJSONLen 字节。
func updateSettingsJSON(core *model.PlayerExtra, value *pb.ObjectValue) (*pb.ObjectValue, error) {
	raw := strings.TrimSpace(value.GetStringValue())
	if raw == "" || len(raw) > consts.MaxSettingsJSONLen {
		return nil, errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "个人设置长度不合法")
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &obj); err != nil || obj == nil {
		return nil, errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "个人设置必须是 JSON 对象")
	}
	core.SettingsJSON = raw
	return &pb.ObjectValue{Kind: &pb.ObjectValue_StringValue{StringValue: core.SettingsJSON}}, nil
}
