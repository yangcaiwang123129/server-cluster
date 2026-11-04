package logic

import (
	"context"
	"sort"

	"server-cluster/internal/cache"
	"server-cluster/internal/record"
	"server-cluster/node/game/app/global"
	"server-cluster/share/convert"
	"server-cluster/share/enum"
	"server-cluster/share/errs"
	"server-cluster/share/model"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
	"server-cluster/share/tpl"
	"server-cluster/share/util"
)

// heroCache 玩家英雄持有（player_hero，一玩家一行，英雄 map JSON）；读走 LRU，写异步批量 upsert。
var heroCache = cache.NewPlayerTableCache[model.PlayerHero](cache.CacherConfig{Name: enum.TablePlayerHero.String()})

type heroLogic struct{}

var heroLogicInst = singleton.NewContainer[global.HeroLogic]("HeroLogic")

func init() {
	heroLogicInst.Register(&heroLogic{})
	registerAsset(&heroAssetHandler{})
}

// Hero 获取英雄业务单例。
func Hero() global.HeroLogic { return heroLogicInst.MustGet() }

// HeroList 英雄列表：一次下发英雄及其绑定的武器/技能/装备（各模块分表，但客户端一次拿全）。
// 静态属性仍由客户端按各自的配置ID关联配置表。
func (this *heroLogic) HeroList(ctx context.Context, playerID int64) (*pb.HeroListResp, error) {
	entries, err := heroEntries(ctx, playerID)
	if err != nil {
		return nil, err
	}
	return &pb.HeroListResp{HeroEntries: entries}, nil
}

// heroEntries 组装英雄完整持有视图：武器/技能按英雄配置的固定绑定反查，装备按所属英雄归组（仅已穿戴）。
// 客户端以本结构为单位刷新，武器/技能/装备任一变化后重新下发即可整体替换。
func heroEntries(ctx context.Context, playerID int64) ([]*pb.HeroEntry, error) {
	row, err := Hero().Get(ctx, playerID)
	if err != nil {
		return nil, err
	}

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

	weaponByCfg := make(map[int32]*pb.WeaponEntry, len(weapons))
	for _, w := range weapons {
		weaponByCfg[w.GetCfgId()] = w
	}
	skillByCfg := make(map[int32]*pb.SkillEntry, len(skills))
	for _, s := range skills {
		skillByCfg[s.GetCfgId()] = s
	}
	equipByHero := make(map[int32][]*pb.EquipEntry, len(equips))
	for _, g := range equips {
		// 客户端只关心已穿戴的装备，未穿戴的不随英雄下发
		if !g.GetEquipped() {
			continue
		}
		equipByHero[g.GetHeroId()] = append(equipByHero[g.GetHeroId()], g)
	}

	entries := make([]*pb.HeroEntry, 0, row.Heroes.Len())
	row.Heroes.Range(func(_ int32, h model.HeroInfo) bool {
		entry := convert.HeroEntry(h)
		if cfg := tpl.T().RoleInfo.Get(h.HeroID); cfg != nil {
			if cfg.WeaponId != 0 {
				entry.Weapon = weaponByCfg[cfg.WeaponId]
			}
			for _, skillID := range cfg.SkillID {
				if skill := skillByCfg[skillID]; skill != nil {
					entry.Skills = append(entry.Skills, skill)
				}
			}
		}
		entry.Equips = equipByHero[h.HeroID]
		entries = append(entries, entry)
		return true
	})
	// map 遍历无序，按配置ID排序保证客户端展示稳定
	sort.Slice(entries, func(i, j int) bool { return entries[i].CfgId < entries[j].CfgId })
	return entries, nil
}

// heroEntry 单个英雄的完整持有视图，用于变更响应与英雄通知。
func heroEntry(ctx context.Context, playerID int64, heroID int32) (*pb.HeroEntry, error) {
	entries, err := heroEntries(ctx, playerID)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.CfgId == heroID {
			return entry, nil
		}
	}
	return nil, errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "英雄未拥有")
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
	row, err := Hero().Get(ctx, playerID)
	if err != nil {
		return model.HeroInfo{}, err
	}
	if _, ok := row.Heroes.Get(cfgID); ok {
		return model.HeroInfo{}, errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "英雄已拥有")
	}

	if err := grantHeroBound(ctx, playerID, int64(cfgID)); err != nil {
		return model.HeroInfo{}, err
	}

	h := newHeroInfo(cfgID)
	if !row.AddHero(h) {
		return model.HeroInfo{}, errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "英雄已拥有")
	}
	heroCache.Set(playerID, row)
	record.Record(ctx, model.RecordModuleHero, playerID, model.RecordActionGain, cfgID, 1)
	return h, nil
}

// HeroAdd 新增英雄：配置ID必须存在，已拥有返回错误。
func (this *heroLogic) HeroAdd(ctx context.Context, playerID int64, req *pb.HeroAddReq) (*pb.HeroAddResp, error) {
	cfgID := req.GetCfgId()
	if !tpl.T().RoleInfo.Has(cfgID) {
		return &pb.HeroAddResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "英雄配置不存在")}, nil
	}

	h, err := addHeroWithBound(ctx, playerID, cfgID)
	if err != nil {
		if r := codedResult(err); r != nil {
			return &pb.HeroAddResp{Result: r}, nil
		}
		return nil, err
	}
	entry, err := heroEntry(ctx, playerID, h.HeroID)
	if err != nil {
		if r := codedResult(err); r != nil {
			return &pb.HeroAddResp{Result: r}, nil
		}
		return nil, err
	}
	global.NotifyHero(ctx, playerID, entry)
	return &pb.HeroAddResp{HeroEntry: entry}, nil
}

// HeroUpdate 更新英雄：整体覆盖持有状态，未拥有返回错误。
func (this *heroLogic) HeroUpdate(ctx context.Context, playerID int64, req *pb.HeroUpdateReq) (*pb.HeroUpdateResp, error) {
	row, err := this.Get(ctx, playerID)
	if err != nil {
		return nil, err
	}

	old, ok := row.Heroes.Get(req.GetCfgId())
	if !ok {
		return &pb.HeroUpdateResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "英雄未拥有")}, nil
	}
	h := old
	h.Level = clampDim(req.GetLevel(), 1)
	h.Exp = req.GetExp()
	if h.Exp < 0 {
		h.Exp = 0
	}
	h.Star = clampDim(req.GetStar(), 1)
	h.SkinID = req.GetSkinId()
	if h.SkinID < 0 {
		h.SkinID = 0
	}
	h.Equipped = req.GetEquipped()
	// 出战英雄全局唯一：设置出战时先取消其它英雄的出战状态，记录被取消出战的英雄用于推送
	var unsetHeroIDs []int32
	if h.Equipped {
		_ = row.Heroes.UpdateAll(func(m map[int32]model.HeroInfo) error {
			for id, other := range m {
				if id != h.HeroID && other.Equipped {
					unsetHeroIDs = append(unsetHeroIDs, id)
					other.Equipped = false
					m[id] = other
				}
			}
			m[h.HeroID] = h
			return nil
		})
	} else if !row.UpdateHero(h) {
		return &pb.HeroUpdateResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "英雄未拥有")}, nil
	}

	this.Save(playerID, row)
	entry, err := heroEntry(ctx, playerID, h.HeroID)
	if err != nil {
		if r := codedResult(err); r != nil {
			return &pb.HeroUpdateResp{Result: r}, nil
		}
		return nil, err
	}
	for _, id := range unsetHeroIDs {
		notifyHero(ctx, playerID, id)
	}
	global.NotifyHero(ctx, playerID, entry)
	return &pb.HeroUpdateResp{HeroEntry: entry}, nil
}

// HeroAttribute 查看英雄累计属性；cfg_id 为 0 时取当前出战英雄。
func (this *heroLogic) HeroAttribute(ctx context.Context, playerID int64, req *pb.HeroAttributeReq) (*pb.HeroAttributeResp, error) {
	hero, attr, err := computeHeroAttribute(ctx, playerID, req.GetCfgId())
	if err != nil {
		if r := codedResult(err); r != nil {
			return &pb.HeroAttributeResp{Result: r}, nil
		}
		return nil, err
	}
	entry, err := heroEntry(ctx, playerID, hero.HeroID)
	if err != nil {
		if r := codedResult(err); r != nil {
			return &pb.HeroAttributeResp{Result: r}, nil
		}
		return nil, err
	}
	return &pb.HeroAttributeResp{HeroEntry: entry, Attribute: attr}, nil
}

// HeroUpgrade 英雄升级：消耗配置表升级素材，等级 +1，最多到等级区间上限。
func (this *heroLogic) HeroUpgrade(ctx context.Context, playerID int64, req *pb.HeroUpgradeReq) (*pb.HeroUpgradeResp, error) {
	row, err := this.Get(ctx, playerID)
	if err != nil {
		return nil, err
	}
	hero, ok := row.Heroes.Get(req.GetCfgId())
	if !ok {
		return &pb.HeroUpgradeResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "英雄未拥有")}, nil
	}

	cfg := tpl.T().AttributeRole.Get(hero.HeroID + clampTier(hero.Star))
	if cfg == nil {
		return &pb.HeroUpgradeResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "英雄属性配置不存在")}, nil
	}
	lo, hi := levelBounds(0, 0, false)
	if cfg.Level != nil {
		lo, hi = levelBounds(cfg.Level.X, cfg.Level.Y, true)
	}
	lvl := clampLevel(hero.Level, lo, hi)
	if lvl >= hi {
		return &pb.HeroUpgradeResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "英雄已满级")}, nil
	}

	cost := costAt(cfg.UpgradeUse, cfg.GrowUGMaterial, lvl, lo)
	if cost > 0 {
		if err := Asset().AssetRemove(ctx, playerID, &tpl.Asset{
			Type:  tpl.AssetType_Item,
			Id:    cfg.UpgradeMaterial,
			Count: cost,
		}); err != nil {
			if r := codedResult(err); r != nil {
				return &pb.HeroUpgradeResp{Result: r}, nil
			}
			return nil, err
		}
	}

	hero.Level = lvl + 1
	if !row.UpdateHero(hero) {
		return &pb.HeroUpgradeResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "英雄未拥有")}, nil
	}
	this.Save(playerID, row)

	_, attr, err := computeHeroAttribute(ctx, playerID, hero.HeroID)
	if err != nil {
		if r := codedResult(err); r != nil {
			return &pb.HeroUpgradeResp{Result: r}, nil
		}
		return nil, err
	}
	entry, err := heroEntry(ctx, playerID, hero.HeroID)
	if err != nil {
		if r := codedResult(err); r != nil {
			return &pb.HeroUpgradeResp{Result: r}, nil
		}
		return nil, err
	}
	global.NotifyHero(ctx, playerID, entry)
	return &pb.HeroUpgradeResp{HeroEntry: entry, Attribute: attr}, nil
}

// EnterGame 进入游戏：加载玩家英雄持有，只下发当前出战英雄的完整持有状态。
func (this *heroLogic) OnEnterGame(ctx context.Context, playerID int64, _ *pb.EnterGameReq, resp *pb.EnterGameResp) error {
	entries, err := heroEntries(ctx, playerID)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Equipped {
			resp.HeroEntry = entry
			return nil
		}
	}
	return nil
}

// LeaveGame 离开游戏：英雄持有数据同步落库，并逐出内存缓存。
func (this *heroLogic) OnLeaveGame(ctx context.Context, playerID int64, _ *pb.LeaveGameReq, _ *pb.LeaveGameResp) error {
	if err := flushAndEvict(ctx, heroCache, playerID); err != nil {
		return err
	}
	return nil
}

// LoadPlayer 加载玩家英雄持有；未建档则初始化空 map 并挂入缓存（异步落库）。
func (this *heroLogic) Get(ctx context.Context, playerID int64) (*model.PlayerHero, error) {
	return heroCache.GetOrInit(ctx, playerID,
		func(id int64) *model.PlayerHero {
			return &model.PlayerHero{PlayerID: id, Heroes: util.NewSafeMap[int32, model.HeroInfo]()}
		},
		func(row *model.PlayerHero) {
			if row.Heroes == nil {
				row.Heroes = util.NewSafeMap[int32, model.HeroInfo]()
			}
		})
}

// Save 写回玩家英雄持有并挂入异步落库队列。
func (this *heroLogic) Save(playerID int64, row *model.PlayerHero) {
	heroCache.Set(playerID, row)
}

// Flush 同步落库并逐出缓存（GM 直改 DB 后强制回源）。
func (this *heroLogic) Flush(ctx context.Context, playerID int64) error {
	return flushAndEvict(ctx, heroCache, playerID)
}

// ============ 资产接口实现（AssetHandler，唯一持有型） ============

// heroAssetHandler 英雄资产实现：落在 player_hero。
type heroAssetHandler struct{}

// AssetType 资产类型：英雄。
func (this *heroAssetHandler) AssetType() tpl.AssetType { return tpl.AssetType_Hero }

// Add 发放英雄；已拥有返回错误。
func (this *heroAssetHandler) Add(ctx context.Context, playerID int64, cfgID, _ int32) error {
	if !tpl.T().RoleInfo.Has(cfgID) {
		return errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "英雄配置不存在")
	}
	h, err := addHeroWithBound(ctx, playerID, cfgID)
	if err != nil {
		return err
	}
	notifyHero(ctx, playerID, h.HeroID)
	return nil
}

// Remove 扣除英雄；未拥有返回错误。
func (this *heroAssetHandler) Remove(ctx context.Context, playerID int64, cfgID, _ int32) error {
	row, err := Hero().Get(ctx, playerID)
	if err != nil {
		return err
	}
	if !row.Heroes.Delete(cfgID) {
		return errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "英雄未拥有")
	}

	heroCache.Set(playerID, row)
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
	row, err := Hero().Get(ctx, playerID)
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
	row, err := Hero().Get(ctx, playerID)
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
		return nil, errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "英雄属性配置不存在")
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
	row, err := this.Get(ctx, playerID)
	if err != nil {
		return nil, err
	}
	h, ok := row.Heroes.Get(heroID)
	if !ok {
		return &pb.HeroAttribute{}, nil
	}
	return heroBaseAttribute(h)
}
