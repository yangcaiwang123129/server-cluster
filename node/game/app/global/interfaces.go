// Package global 全局契约与装配：业务接口定义 + 基础设施初始化。
// 依赖方向：handler -> logic -> global；logic 通过包级 init 自注册单例。
package global

import (
	"context"
	"server-cluster/internal/actor"

	"server-cluster/share/model"

	"server-cluster/share/pb"
	"server-cluster/share/tpl"
)

// PlayerLifecycle 玩家生命周期钩子接口
type PlayerLifecycle interface {
	// OnEnterGame 进入游戏
	OnEnterGame(ctx context.Context, playerID int64, req *pb.EnterGameReq, resp *pb.EnterGameResp) error
	// OnLeaveGame 离开游戏
	OnLeaveGame(ctx context.Context, playerID int64, req *pb.LeaveGameReq, resp *pb.LeaveGameResp) error
}

// PlayerModule 玩家缓存模块接口
type PlayerModule[T model.PlayerRow] interface {
	PlayerLifecycle
	// Load 读缓存；未建档则初始化并写入缓存
	Load(ctx context.Context, playerID int64) (T, error)
	// Save 异步入库
	Save(playerID int64, row T)
	// Flush 同步落库该玩家的行并清除缓存
	Flush(ctx context.Context, playerID int64) error
}

// PlayerLogic 玩家模块业务接口
type PlayerLogic interface {
	PlayerModule[*model.PlayerExtra]
	// GetActor 获取玩家actor
	GetActor(ctx context.Context, playerID int64) (*actor.PlayerActor, error)
	// GetActors 玩家获取所有actor
	GetActors(ctx context.Context) ([]*actor.PlayerActor, error)
	// CreateActor 玩家创建actor（verify 首包绑定后调用）；loginIP 由 handler 从 node.Context.GetIP() 取后传入，内存维护
	CreateActor(ctx context.Context, playerID int64, loginIP string) error
	// DestroyActor 玩家销毁actor
	DestroyActor(ctx context.Context, playerID int64) error
	// DestroyActorIfOffline 断线延迟销毁：仅当玩家在 since 之后未重连时才销毁 actor
	DestroyActorIfOffline(ctx context.Context, playerID int64, since int64) error
	// Kick 玩家踢下线
	Kick(ctx context.Context, playerID int64) error
	// GiftBuy 购买礼包（按礼包配置扣除消耗资产并发放奖励）
	GiftBuy(ctx context.Context, playerID int64, req *pb.GiftBuyReq) (*pb.GiftBuyResp, error)
	// Exchange 兑换（按兑换配置扣除消耗资产并发放奖励）
	Exchange(ctx context.Context, playerID int64, req *pb.ExchangeReq) (*pb.ExchangeResp, error)
	// Recharge 充值（当前为假充值，校验配置后直接发放奖励）
	Recharge(ctx context.Context, playerID int64, req *pb.RechargeReq) (*pb.RechargeResp, error)
	// AttrOperate 属性接口：按属性类型执行更新/领取
	AttrOperate(ctx context.Context, playerID int64, req *pb.AttrOperateReq) (*pb.AttrOperateResp, error)
}

// ItemLogic 物品容器业务接口
type ItemLogic interface {
	PlayerModule[*model.PlayerItem]
	// ItemContainerList 物品容器列表
	ItemContainerList(ctx context.Context, playerID int64, req *pb.ItemContainerListReq) (*pb.ItemContainerListResp, error)
	// ItemMove 物品容器内移动物品
	ItemMove(ctx context.Context, playerID int64, req *pb.ItemMoveReq) (*pb.ItemMoveResp, error)
	// ItemRotate 物品容器内旋转物品
	ItemRotate(ctx context.Context, playerID int64, req *pb.ItemRotateReq) (*pb.ItemRotateResp, error)
	// ItemSort 物品容器一键整理
	ItemSort(ctx context.Context, playerID int64, req *pb.ItemSortReq) (*pb.ItemSortResp, error)
	// ItemTransfer 物品容器间移动物品
	ItemTransfer(ctx context.Context, playerID int64, req *pb.ItemTransferReq) (*pb.ItemTransferResp, error)
	// ItemAddEntry 向物品容器放入对象（GM，返回变更后的物品容器）
	ItemAddEntry(ctx context.Context, playerID int64, req *pb.GmItemAddEntryReq) (*pb.ItemContainerListResp, error)
}

// HeroLogic 英雄业务接口
type HeroLogic interface {
	PlayerModule[*model.PlayerHero]
	// HeroList 英雄列表
	HeroList(ctx context.Context, playerID int64) (*pb.HeroListResp, error)
	// HeroAdd 新增英雄（获得英雄时由武器/技能模块按配置初始化固定绑定的武器/技能）
	HeroAdd(ctx context.Context, playerID int64, req *pb.HeroAddReq) (*pb.HeroAddResp, error)
	// HeroUpdate 更新英雄
	HeroUpdate(ctx context.Context, playerID int64, req *pb.HeroUpdateReq) (*pb.HeroUpdateResp, error)
	// HeroAttribute 查看英雄累计属性（cfg_id 为 0 表示出战英雄）
	HeroAttribute(ctx context.Context, playerID int64, req *pb.HeroAttributeReq) (*pb.HeroAttributeResp, error)
	// HeroUpgrade 英雄升级（消耗物品）
	HeroUpgrade(ctx context.Context, playerID int64, req *pb.HeroUpgradeReq) (*pb.HeroUpgradeResp, error)
	// AttributeBonus 英雄本体属性贡献（未持有返回零值，不填充 Power）
	AttributeBonus(ctx context.Context, playerID int64, heroID int32) (*pb.HeroAttribute, error)
}

// WeaponLogic 武器模块接口
type WeaponLogic interface {
	PlayerModule[*model.PlayerWeapon]
	// WeaponList 武器持有列表（供 HeroList 聚合下发与 GM 查询）
	WeaponList(ctx context.Context, playerID int64) ([]*pb.WeaponEntry, error)
	// WeaponUpdate 更新武器
	WeaponUpdate(ctx context.Context, playerID int64, req *pb.WeaponUpdateReq) (*pb.WeaponUpdateResp, error)
	// WeaponUpgrade 武器升级（消耗升级素材）
	WeaponUpgrade(ctx context.Context, playerID int64, req *pb.WeaponUpgradeReq) (*pb.WeaponUpgradeResp, error)
	// WeaponUpQuality 武器升品（消耗升品素材）
	WeaponUpQuality(ctx context.Context, playerID int64, req *pb.WeaponUpQualityReq) (*pb.WeaponUpQualityResp, error)
	// WeaponBreakthrough 武器突破（消耗突破素材）
	WeaponBreakthrough(ctx context.Context, playerID int64, req *pb.WeaponBreakthroughReq) (*pb.WeaponBreakthroughResp, error)
	// WeaponEnsureForHero 按英雄配置初始化固定绑定武器（获得英雄/进游戏兜底）
	WeaponEnsureForHero(ctx context.Context, playerID int64, heroCfgID int32) error
	// AttributeBonus 武器对英雄的伤害贡献（英雄未绑定武器或未持有时为零值，不填充 Power）
	AttributeBonus(ctx context.Context, playerID int64, heroID int32) (*pb.HeroAttribute, error)
}

// SkillLogic 技能模块接口
type SkillLogic interface {
	PlayerModule[*model.PlayerSkill]
	// SkillList 技能持有列表（供 HeroList 聚合下发与 GM 查询）
	SkillList(ctx context.Context, playerID int64) ([]*pb.SkillEntry, error)
	// SkillUpdate 更新技能
	SkillUpdate(ctx context.Context, playerID int64, req *pb.SkillUpdateReq) (*pb.SkillUpdateResp, error)
	// SkillUpgrade 技能升级（消耗升级素材）
	SkillUpgrade(ctx context.Context, playerID int64, req *pb.SkillUpgradeReq) (*pb.SkillUpgradeResp, error)
	// SkillUpQuality 技能升品（消耗升品素材）
	SkillUpQuality(ctx context.Context, playerID int64, req *pb.SkillUpQualityReq) (*pb.SkillUpQualityResp, error)
	// SkillBreakthrough 技能突破（消耗突破素材）
	SkillBreakthrough(ctx context.Context, playerID int64, req *pb.SkillBreakthroughReq) (*pb.SkillBreakthroughResp, error)
	// SkillEnsureForHero 按英雄配置初始化固定绑定技能（获得英雄/进游戏兜底）
	SkillEnsureForHero(ctx context.Context, playerID int64, heroCfgID int32) error
}

// EquipLogic 装备模块接口
type EquipLogic interface {
	PlayerModule[*model.PlayerEquip]
	// EquipList 装备持有列表（供 HeroList 聚合下发与 GM 查询）
	EquipList(ctx context.Context, playerID int64) ([]*pb.EquipEntry, error)
	// EquipAdd 新增装备
	EquipAdd(ctx context.Context, playerID int64, req *pb.EquipAddReq) (*pb.EquipAddResp, error)
	// EquipUpdate 更新装备
	EquipUpdate(ctx context.Context, playerID int64, req *pb.EquipUpdateReq) (*pb.EquipUpdateResp, error)
	// EquipWear 穿戴装备（同英雄同配置自动顶替）
	EquipWear(ctx context.Context, playerID int64, req *pb.EquipWearReq) (*pb.EquipWearResp, error)
	// EquipUnwear 卸载装备
	EquipUnwear(ctx context.Context, playerID int64, req *pb.EquipUnwearReq) (*pb.EquipUnwearResp, error)
	// EquipUpgrade 装备升级（消耗升级素材）
	EquipUpgrade(ctx context.Context, playerID int64, req *pb.EquipUpgradeReq) (*pb.EquipUpgradeResp, error)
	// EquipUpQuality 装备升品（消耗升品素材）
	EquipUpQuality(ctx context.Context, playerID int64, req *pb.EquipUpQualityReq) (*pb.EquipUpQualityResp, error)
	// EquipBreakthrough 装备突破（消耗突破素材）
	EquipBreakthrough(ctx context.Context, playerID int64, req *pb.EquipBreakthroughReq) (*pb.EquipBreakthroughResp, error)
	// AttributeBonus 穿戴在该英雄身上的装备对生命/护甲的贡献（不填充 Power）
	AttributeBonus(ctx context.Context, playerID int64, heroID int32) (*pb.HeroAttribute, error)
}

// MailLogic 邮件模块接口
type MailLogic interface {
	PlayerModule[*model.PlayerMail]
	// MailList 邮件列表（未读优先，按创建时间倒序；含未读数用于红点）
	MailList(ctx context.Context, playerID int64) (*pb.MailListResp, error)
	// MailRead 标记单封已读
	MailRead(ctx context.Context, playerID int64, req *pb.MailReadReq) (*pb.MailReadResp, error)
	// MailReadAll 全部标记已读
	MailReadAll(ctx context.Context, playerID int64, req *pb.MailReadAllReq) (*pb.MailReadAllResp, error)
	// MailClaim 领取单封附件（发放资产并标记已领取）
	MailClaim(ctx context.Context, playerID int64, req *pb.MailClaimReq) (*pb.MailClaimResp, error)
	// MailClaimAll 一键领取全部未领取附件
	MailClaimAll(ctx context.Context, playerID int64, req *pb.MailClaimAllReq) (*pb.MailClaimAllResp, error)
	// MailDelete 删除单封邮件（奖励邮件未领取禁止删除）
	MailDelete(ctx context.Context, playerID int64, req *pb.MailDeleteReq) (*pb.MailDeleteResp, error)
	// MailDeleteClean 清理全部可删除邮件（已读/已领取，或无附件）
	MailDeleteClean(ctx context.Context, playerID int64, req *pb.MailDeleteCleanReq) (*pb.MailDeleteCleanResp, error)
	// MailSend 后台发送邮件（可带附件），供 GM 调用
	MailSend(ctx context.Context, playerID int64, req *pb.GmMailSendReq) (*pb.MailEntry, error)
}

// TaskEvent 任务进度事件：由业务埋点上报，按 targetTaskType + param 匹配任务配置。
type TaskEvent struct {
	PlayerID       int64 // 玩家ID
	TargetTaskType int32 // 进度来源类型（tpl.TargetTaskType_*）
	Param          int32 // 细分参数（如充值配置ID），0 匹配全部
	Count          int64 // 增量
}

// TaskLogic 任务模块接口
type TaskLogic interface {
	PlayerModule[*model.PlayerTask]
	// TaskList 任务列表（按类型/活动过滤，含进度与红点）
	TaskList(ctx context.Context, playerID int64, req *pb.TaskListReq) (*pb.TaskListResp, error)
	// TaskClaim 任务领奖
	TaskClaim(ctx context.Context, playerID int64, req *pb.TaskClaimReq) (*pb.TaskClaimResp, error)
	// TaskAccept 任务接取（仅手动接取任务：未接取→进行中）
	TaskAccept(ctx context.Context, playerID int64, req *pb.TaskAcceptReq) (*pb.TaskAcceptResp, error)
	// EmitTaskEvent 上报任务进度事件（内部吞掉错误，不阻断业务主流程）
	EmitTaskEvent(ctx context.Context, e TaskEvent)
}

// AssetLogic 资产接口
type AssetLogic interface {
	// AssetAdd 发放资产；Count<=0 按 1 处理
	AssetAdd(ctx context.Context, playerID int64, asset *tpl.Asset) error
	// AssetRemove 扣除资产；持有不足返回错误
	AssetRemove(ctx context.Context, playerID int64, asset *tpl.Asset) error
	// AssetUpdate 覆盖持有状态：Count<=0 表示移除，>0 表示确保持有（可叠加类型按数量覆盖）
	AssetUpdate(ctx context.Context, playerID int64, asset *tpl.Asset) error
	// AssetQuery 查询单个配置ID的持有数量；未持有返回 0
	AssetQuery(ctx context.Context, playerID int64, assetType tpl.AssetType, cfgID int32) (int32, error)
	// AssetList 列出某类资产的全部持有
	AssetList(ctx context.Context, playerID int64, assetType tpl.AssetType) ([]*tpl.Asset, error)
}

// AssetStore 单类资产的增删改查接口
type AssetStore interface {
	// AssetType 该实现负责的资产类型
	AssetType() tpl.AssetType
	// Add 发放；Count<=0 按 1 处理
	Add(ctx context.Context, playerID int64, cfgID, count int32) error
	// Remove 扣除；持有不足返回错误
	Remove(ctx context.Context, playerID int64, cfgID, count int32) error
	// Update 覆盖持有：count<=0 移除，>0 确保持有
	Update(ctx context.Context, playerID int64, cfgID, count int32) error
	// Query 查询持有数量
	Query(ctx context.Context, playerID int64, cfgID int32) (int32, error)
	// List 列出该类资产全部持有
	List(ctx context.Context, playerID int64) ([]*tpl.Asset, error)
}
