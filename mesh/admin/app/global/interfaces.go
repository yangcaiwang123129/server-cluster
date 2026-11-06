package global

import (
	"context"

	"server-cluster/share/dto"
	"server-cluster/share/model"
)

// ActivityLogic admin 内嵌活动模块：活动状态机 + 开服策略 + 快照发布 + GM 操作。
type ActivityLogic interface {
	// Start 加载活动实例与开服时间，修复跨停机状态，发布首份快照并启动 ticker。
	Start(ctx context.Context) error
	// Stop 停止 ticker。
	Stop()
	// List GM 分页查询活动实例。
	List(ctx context.Context, q dto.GmActivityQuery, pageIndex, pageSize int) ([]dto.GmActivityResp, int64, error)
	// Configs GM 活动模板配置展示。
	Configs() []dto.GmActivityCfgResp
	// UpdateTime GM 修改活动时间（仅未开启允许）。
	UpdateTime(ctx context.Context, id int64, req dto.GmActivityTimeReq) (*dto.GmActivityResp, error)
	// End GM 提前结束活动。
	End(ctx context.Context, id int64, operator string) (*dto.GmActivityResp, error)
	// Snapshot 获取当前内存快照（只读）。
	Snapshot() *dto.ActivitySnapshot
}

// ClusterLogic 集群管理业务接口。
//
// 节点数据源：每次列表前从 etcd 拉取当前在线实例 → 与 gm_cluster_node 表做 upsert。
// 远程运维：通过 SSH 执行服务器上的 start.sh / stop.sh。
type ClusterLogic interface {
	// Sync 从 etcd 同步全量节点到 gm_cluster_node（幂等，可重复调用）。
	Sync(ctx context.Context) error
	// List 分页查询集群节点（列表前自动触发 Sync）。
	List(ctx context.Context, q dto.ClusterNodeQuery, pageIndex, pageSize int) ([]dto.ClusterNodeResp, int64, error)
	// Get 查询单个节点详情。
	Get(ctx context.Context, nodeID string) (*dto.ClusterNodeResp, error)
	// UpdateSSH 更新单个节点的运维侧 SSH 连接配置。
	UpdateSSH(ctx context.Context, m *model.ClusterNode) error
	// Exec 在目标服务器上执行运维脚本并回显输出。
	// action: start / stop / restart / deploy。
	Exec(ctx context.Context, nodeID, action string) (*dto.ClusterExecResp, error)
}

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

// CatalogLogic 游戏配置目录业务接口（英雄/装备/物品，选择器与列表共用数据源）。
type CatalogLogic interface {
	// Catalog 读 Luban 配置生成目录；typ: hero|equip|item。
	// 全部类型均支持分页：pageSize>0 返回当页数据与 total，pageSize<=0 返回全量。
	Catalog(typ, keyword string, pageIndex, pageSize int) ([]dto.GameCatalogEntry, int64, error)
}

// GmHeroLogic 后台 GM 英雄模块业务接口。
type GmHeroLogic interface {
	// HeroList 英雄列表（含绑定的武器/技能/装备）
	HeroList(req dto.GmPlayerReq) (*dto.GmHeroListResp, error)
	// HeroAdd 新增英雄
	HeroAdd(req dto.GmHeroAddReq) (*dto.GmHeroEntry, error)
	// HeroUpdate 更新英雄
	HeroUpdate(req dto.GmHeroUpdateReq) (*dto.GmHeroEntry, error)
	// HeroAttribute 查看英雄累计属性
	HeroAttribute(req dto.GmHeroAttributeReq) (*dto.GmHeroAttribute, error)
	// HeroUpgrade 英雄升级（消耗物品）
	HeroUpgrade(req dto.GmUpgradeReq) (*dto.GmUpgradeResp, error)
}

// GmEquipLogic 后台 GM 装备模块业务接口。
type GmEquipLogic interface {
	// EquipList 装备列表
	EquipList(req dto.GmPlayerReq) ([]dto.GmEquipEntry, error)
	// EquipAdd 新增装备
	EquipAdd(req dto.GmEquipAddReq) (*dto.GmEquipEntry, error)
	// EquipUpdate 更新装备
	EquipUpdate(req dto.GmEquipUpdateReq) (*dto.GmEquipEntry, error)
	// EquipWear 装备穿戴（同英雄同配置自动顶替）
	EquipWear(req dto.GmEquipWearReq) (*dto.GmEquipEntry, error)
	// EquipUnWear 装备卸载
	EquipUnWear(req dto.GmEquipUnWearReq) (*dto.GmEquipEntry, error)
	// EquipUpgrade 装备升级（消耗物品）
	EquipUpgrade(req dto.GmUpgradeReq) (*dto.GmUpgradeResp, error)
	// EquipUpQuality 装备升品（消耗升品素材）
	EquipUpQuality(req dto.GmUpgradeReq) (*dto.GmUpgradeResp, error)
	// EquipBreakthrough 装备突破（消耗突破素材）
	EquipBreakthrough(req dto.GmUpgradeReq) (*dto.GmUpgradeResp, error)
}

// GmItemLogic 后台 GM 物品容器模块业务接口。
type GmItemLogic interface {
	// ItemContainerList 物品容器列表
	ItemContainerList(req dto.GmItemListReq) ([]dto.GmItemResp, error)
	// ItemMove 物品容器内移动物品
	ItemMove(req dto.GmItemMoveReq) error
	// ItemRotate 物品容器内旋转物品
	ItemRotate(req dto.GmItemRotateReq) error
	// ItemSort 物品容器一键整理
	ItemSort(req dto.GmItemSortReq) error
	// ItemTransfer 物品容器间移动物品（返回变更后的源/目标容器）
	ItemTransfer(req dto.GmItemTransferReq) ([]dto.GmItemResp, error)
	// ItemAddEntry 向物品容器放入对象（服务端按"从左往右、从上到下"自动落位，返回变更后的容器）
	ItemAddEntry(req dto.GmItemAddEntryReq) ([]dto.GmItemResp, error)
	// ItemClear 清除物品（传实例ID精确清除单个；不传实例ID清空指定容器，容器类型为 0 时全清）
	ItemClear(req dto.GmItemClearReq) ([]dto.GmItemResp, error)
}

// GmMa	ilLogic 后台 GM 邮件模块业务接口。
type GmMailLogic interface {
	// MailList 玩家邮件列表
	MailList(req dto.GmPlayerReq) ([]dto.GmMailEntry, error)
	// MailSend 发送邮件（可带附件）
	MailSend(req dto.GmMailSendReq) (*dto.GmMailEntry, error)
}

// GmRecordLogic 后台打点记录查询业务接口。
type GmRecordLogic interface {
	// RecordList 分页查询某模块的获得/消耗记录，并回填配置展示信息。
	RecordList(q dto.GmRecordQuery, pageIndex, pageSize int) ([]dto.GmRecordEntry, int64, error)
}
