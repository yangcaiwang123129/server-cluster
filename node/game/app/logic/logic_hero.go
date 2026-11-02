package logic

import (
	"context"
	"errors"
	"server-cluster/internal/record"
	"sort"

	"gorm.io/gorm"

	"server-cluster/node/game/app/global"
	"server-cluster/share/errs"
	"server-cluster/share/model"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
	"server-cluster/share/tpl"
	"server-cluster/share/util"
)

type heroLogic struct{}

var heroLogicInst = singleton.NewContainer[global.HeroLogic]("HeroLogic")

func init() {
	heroLogicInst.Register(&heroLogic{})
	registerAsset(&heroAssetHandler{})
}

// Hero 获取英雄业务单例。
func Hero() global.HeroLogic { return heroLogicInst.MustGet() }

// HeroList 英雄列表：服务器按模块分表，但客户端一次拿全（英雄 + 武器 + 技能 + 装备）。
// 静态属性仍由客户端按各自的配置ID关联配置表。
func (this *heroLogic) HeroList(ctx context.Context, playerID int64) (*pb.HeroListResp, error) {
	row, err := loadPlayerHero(ctx, playerID)
	if err != nil {
		return nil, err
	}

	entries := make([]*pb.HeroEntry, 0, row.Heroes.Len())
	row.Heroes.Range(func(_ int32, h model.HeroInfo) bool {
		entries = append(entries, toHeroEntry(h))
		return true
	})
	// map 遍历无序，按配置ID排序保证客户端展示稳定
	sort.Slice(entries, func(i, j int) bool { return entries[i].CfgId < entries[j].CfgId })

	weapons, err := Weapon().WeaponList(ctx, playerID)
	if err != nil {
		return nil, err
	}
	skills, err := Skill().SkillList(ctx, playerID)
	if err != nil {
		return nil, err
	}
	equips, err := Equip().EquipList(ctx, playerID)
	if err != nil {
		return nil, err
	}

	return &pb.HeroListResp{
		HeroEntries:   entries,
		WeaponEntries: weapons,
		SkillEntries:  skills,
		EquipEntries:  equips,
	}, nil
}

// toHeroEntry 持有数据 → wire 条目（仅英雄状态；武器/技能/装备随 HeroList 一并下发）。
func toHeroEntry(h model.HeroInfo) *pb.HeroEntry {
	return &pb.HeroEntry{
		CfgId:    h.HeroID,
		Level:    h.Level,
		Exp:      h.Exp,
		Star:     h.Star,
		SkinId:   h.SkinID,
		Equipped: h.Equipped,
	}
}

// newHeroInfo 按英雄配置初始化持有状态。
func newHeroInfo(cfgID int32) model.HeroInfo {
	return model.HeroInfo{HeroID: cfgID}
}

// grantHeroBound 获得英雄后按配置初始化固定绑定的武器/技能（独立模块）。
func grantHeroBound(ctx context.Context, playerID, heroCfgID int64) error {
	if err := Weapon().WeaponEnsureForHero(ctx, playerID, int32(heroCfgID)); err != nil {
		return err
	}
	return Skill().SkillEnsureForHero(ctx, playerID, int32(heroCfgID))
}

// addHeroWithBound 新增英雄并初始化其固定绑定的武器/技能。
// 先建武器/技能、再写入英雄，避免初始化失败后英雄已落进内存缓存造成脏数据。
func addHeroWithBound(ctx context.Context, playerID int64, cfgID int32) (model.HeroInfo, error) {
	row, err := loadPlayerHero(ctx, playerID)
	if err != nil {
		return model.HeroInfo{}, err
	}
	if _, ok := row.Heroes.Get(cfgID); ok {
		return model.HeroInfo{}, errs.CodeInvalidParams.WithMessage("英雄已拥有").Err()
	}

	if err := grantHeroBound(ctx, playerID, int64(cfgID)); err != nil {
		return model.HeroInfo{}, err
	}

	h := newHeroInfo(cfgID)
	if !row.AddHero(h) {
		return model.HeroInfo{}, errs.CodeInvalidParams.WithMessage("英雄已拥有").Err()
	}
	global.PlayerHeroCache.Set(playerID, row)
	record.Record(ctx, model.RecordModuleHero, playerID, model.RecordActionGain, cfgID, 1)
	return h, nil
}

// HeroAdd 新增英雄：配置ID必须存在，已拥有返回错误。
func (this *heroLogic) HeroAdd(ctx context.Context, playerID int64, req *pb.HeroAddReq) (*pb.HeroAddResp, error) {
	cfgID := req.GetCfgId()
	if !tpl.T().RoleInfo.Has(cfgID) {
		return nil, errs.CodeInvalidParams.WithMessage("英雄配置不存在").Err()
	}

	h, err := addHeroWithBound(ctx, playerID, cfgID)
	if err != nil {
		return nil, err
	}
	return &pb.HeroAddResp{HeroEntry: toHeroEntry(h)}, nil
}

// HeroUpdate 更新英雄：整体覆盖持有状态，未拥有返回错误。
func (this *heroLogic) HeroUpdate(ctx context.Context, playerID int64, req *pb.HeroUpdateReq) (*pb.HeroUpdateResp, error) {
	row, err := loadPlayerHero(ctx, playerID)
	if err != nil {
		return nil, err
	}

	old, ok := row.Heroes.Get(req.GetCfgId())
	if !ok {
		return nil, errs.CodeInvalidParams.WithMessage("英雄未拥有").Err()
	}
	h := old
	h.Level = req.GetLevel()
	h.Exp = req.GetExp()
	h.Star = req.GetStar()
	h.SkinID = req.GetSkinId()
	h.Equipped = req.GetEquipped()
	// 出战英雄全局唯一：设置出战时先取消其它英雄的出战状态
	if h.Equipped {
		_ = row.Heroes.UpdateAll(func(m map[int32]model.HeroInfo) error {
			for id, other := range m {
				if id != h.HeroID && other.Equipped {
					other.Equipped = false
					m[id] = other
				}
			}
			m[h.HeroID] = h
			return nil
		})
	} else if !row.UpdateHero(h) {
		return nil, errs.CodeInvalidParams.WithMessage("英雄未拥有").Err()
	}

	global.PlayerHeroCache.Set(playerID, row)
	return &pb.HeroUpdateResp{HeroEntry: toHeroEntry(h)}, nil
}

// HeroAttribute 查看英雄累计属性；cfg_id 为 0 时取当前出战英雄。
func (this *heroLogic) HeroAttribute(ctx context.Context, playerID int64, req *pb.HeroAttributeReq) (*pb.HeroAttributeResp, error) {
	hero, attr, err := computeHeroAttribute(ctx, playerID, req.GetCfgId())
	if err != nil {
		return nil, err
	}
	return &pb.HeroAttributeResp{HeroEntry: toHeroEntry(*hero), Attribute: attr}, nil
}

// HeroUpgrade 英雄升级：消耗配置表升级素材，等级 +1，最多到等级区间上限。
func (this *heroLogic) HeroUpgrade(ctx context.Context, playerID int64, req *pb.HeroUpgradeReq) (*pb.HeroUpgradeResp, error) {
	row, err := loadPlayerHero(ctx, playerID)
	if err != nil {
		return nil, err
	}
	hero, ok := row.Heroes.Get(req.GetCfgId())
	if !ok {
		return nil, errs.CodeInvalidParams.WithMessage("英雄未拥有").Err()
	}

	cfg := tpl.T().AttributeRole.Get(hero.HeroID + clampTier(hero.Star))
	if cfg == nil {
		return nil, errs.CodeInvalidParams.WithMessage("英雄属性配置不存在").Err()
	}
	lo, hi := levelBounds(0, 0, false)
	if cfg.Level != nil {
		lo, hi = levelBounds(cfg.Level.X, cfg.Level.Y, true)
	}
	lvl := clampLevel(hero.Level, lo, hi)
	if lvl >= hi {
		return nil, errs.CodeInvalidParams.WithMessage("英雄已满级").Err()
	}

	cost := costAt(cfg.UpgradeUse, cfg.GrowUGMaterial, lvl, lo)
	if cost > 0 {
		if err := Asset().AssetRemove(ctx, playerID, &tpl.Asset{
			Type:  tpl.AssetType_Item,
			Id:    cfg.UpgradeMaterial,
			Count: cost,
		}); err != nil {
			return nil, err
		}
	}

	hero.Level = lvl + 1
	if !row.UpdateHero(hero) {
		return nil, errs.CodeInvalidParams.WithMessage("英雄未拥有").Err()
	}
	global.PlayerHeroCache.Set(playerID, row)

	_, attr, err := computeHeroAttribute(ctx, playerID, hero.HeroID)
	if err != nil {
		return nil, err
	}
	return &pb.HeroUpgradeResp{HeroEntry: toHeroEntry(hero), Attribute: attr}, nil
}

// EnterGame 进入游戏：加载玩家英雄持有，未建档则初始化空集合。
func (this *heroLogic) EnterGame(ctx context.Context, playerID int64, _ *pb.EnterGameReq) (*pb.EnterGameResp, error) {
	row, err := loadPlayerHero(ctx, playerID)
	if err != nil {
		return nil, err
	}
	// 只下发当前出战英雄的状态，静态属性由客户端关联英雄配置表
	return &pb.EnterGameResp{HeroEntry: equippedHeroEntry(row)}, nil
}

// equippedHeroEntry 当前出战英雄；未出战返回 nil。
func equippedHeroEntry(row *model.PlayerHero) *pb.HeroEntry {
	if row.Heroes == nil {
		return nil
	}
	var entry *pb.HeroEntry
	row.Heroes.Range(func(_ int32, h model.HeroInfo) bool {
		if !h.Equipped {
			return true
		}
		entry = toHeroEntry(h)
		return false
	})
	return entry
}

// LeaveGame 离开游戏：英雄持有数据同步落库，并逐出内存缓存。
func (this *heroLogic) LeaveGame(ctx context.Context, playerID int64, _ *pb.LeaveGameReq) (*pb.LeaveGameResp, error) {
	if err := flushAndEvict(ctx, global.PlayerHeroCache, playerID); err != nil {
		return nil, err
	}
	return &pb.LeaveGameResp{}, nil
}

// loadPlayerHero 加载玩家英雄持有；未建档则初始化空 map 并挂入缓存（异步落库）。
func loadPlayerHero(ctx context.Context, playerID int64) (*model.PlayerHero, error) {
	row, err := global.PlayerHeroCache.Get(ctx, playerID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row = &model.PlayerHero{PlayerID: playerID, Heroes: util.NewSafeMap[int32, model.HeroInfo]()}
		global.PlayerHeroCache.Set(playerID, row)
		return row, nil
	}
	if err != nil {
		return nil, err
	}
	// DB 中该列为 NULL 时字段为 nil，补成空集合避免业务写操作 panic
	if row.Heroes == nil {
		row.Heroes = util.NewSafeMap[int32, model.HeroInfo]()
	}
	return row, nil
}

// ============ 资产接口实现（AssetHandler，唯一持有型） ============

// heroAssetHandler 英雄资产实现：落在 player_hero。
type heroAssetHandler struct{}

// AssetType 资产类型：英雄。
func (this *heroAssetHandler) AssetType() tpl.AssetType { return tpl.AssetType_Hero }

// Add 发放英雄；已拥有返回错误。
func (this *heroAssetHandler) Add(ctx context.Context, playerID int64, cfgID, _ int32) error {
	if !tpl.T().RoleInfo.Has(cfgID) {
		return errs.CodeInvalidParams.WithMessage("英雄配置不存在").Err()
	}
	_, err := addHeroWithBound(ctx, playerID, cfgID)
	return err
}

// Remove 扣除英雄；未拥有返回错误。
func (this *heroAssetHandler) Remove(ctx context.Context, playerID int64, cfgID, _ int32) error {
	row, err := loadPlayerHero(ctx, playerID)
	if err != nil {
		return err
	}
	if !row.Heroes.Delete(cfgID) {
		return errs.CodeInvalidParams.WithMessage("英雄未拥有").Err()
	}

	global.PlayerHeroCache.Set(playerID, row)
	record.Record(ctx, model.RecordModuleHero, playerID, model.RecordActionConsume, cfgID, 1)
	return nil
}

// Update 覆盖持有：count<=0 移除，>0 确保持有。
func (this *heroAssetHandler) Update(ctx context.Context, playerID int64, cfgID, count int32) error {
	owned, err := this.Query(ctx, playerID, cfgID)
	if err != nil {
		return err
	}
	switch {
	case count <= 0 && owned > 0:
		return this.Remove(ctx, playerID, cfgID, 1)
	case count > 0 && owned == 0:
		return this.Add(ctx, playerID, cfgID, 1)
	default:
		return nil
	}
}

// Query 查询是否持有：持有 1，未持有 0。
func (this *heroAssetHandler) Query(ctx context.Context, playerID int64, cfgID int32) (int32, error) {
	row, err := loadPlayerHero(ctx, playerID)
	if err != nil {
		return 0, err
	}
	if _, ok := row.Heroes.Get(cfgID); !ok {
		return 0, nil
	}
	return 1, nil
}

// List 列出全部英雄（按配置ID升序）。
func (this *heroAssetHandler) List(ctx context.Context, playerID int64) ([]*tpl.Asset, error) {
	row, err := loadPlayerHero(ctx, playerID)
	if err != nil {
		return nil, err
	}

	assets := make([]*tpl.Asset, 0, row.Heroes.Len())
	row.Heroes.Range(func(cfgID int32, _ model.HeroInfo) bool {
		assets = append(assets, &tpl.Asset{Type: tpl.AssetType_Hero, Id: cfgID, Count: 1})
		return true
	})
	sort.Slice(assets, func(i, j int) bool { return assets[i].Id < assets[j].Id })
	return assets, nil
}

// heroBaseAttribute 英雄本体属性：星级档位决定 AttributeRole 行，等级走加法成长。
func heroBaseAttribute(h model.HeroInfo) (*pb.HeroAttribute, error) {
	cfg := tpl.T().AttributeRole.Get(h.HeroID + clampTier(h.Star))
	if cfg == nil {
		return nil, errs.CodeInvalidParams.WithMessage("英雄属性配置不存在").Err()
	}
	lo, hi := levelBounds(0, 0, false)
	if cfg.Level != nil {
		lo, hi = levelBounds(cfg.Level.X, cfg.Level.Y, true)
	}
	lvl := clampLevel(h.Level, lo, hi)
	return &pb.HeroAttribute{
		Hp:     statAt(cfg.Hp, cfg.GrowHp, lvl, lo),
		Armor:  statAt(cfg.Armor, cfg.GrowArmor, lvl, lo),
		Damage: statAt(cfg.Damage, cfg.GrowDamage, lvl, lo),
	}, nil
}

// AttributeBonus 英雄本体属性贡献（未持有返回零值）。
func (this *heroLogic) AttributeBonus(ctx context.Context, playerID int64, heroID int32) (*pb.HeroAttribute, error) {
	row, err := loadPlayerHero(ctx, playerID)
	if err != nil {
		return nil, err
	}
	h, ok := row.Heroes.Get(heroID)
	if !ok {
		return &pb.HeroAttribute{}, nil
	}
	return heroBaseAttribute(h)
}
