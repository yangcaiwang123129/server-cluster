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

// weaponCache 玩家武器持有（player_weapon，一玩家一行，武器 map JSON）；读走 LRU，写异步批量 upsert。
var weaponCache = cache.NewPlayerTableCache[model.PlayerWeapon](cache.CacherConfig{Name: enum.TablePlayerWeapon.String()})

type weaponLogic struct{}

var weaponLogicInst = singleton.NewContainer[global.WeaponLogic]("WeaponLogic")

func init() {
	weaponLogicInst.Register(&weaponLogic{})
	registerAsset(&weaponAssetStore{})
}

// Weapon 获取武器业务单例。
func Weapon() global.WeaponLogic { return weaponLogicInst.MustGet() }

// newWeaponInfo 新获得武器的默认状态：等级 1、品质 1、突破 0。
func newWeaponInfo(cfgID int32) model.WeaponInfo {
	return model.WeaponInfo{WeaponID: cfgID, Level: 1, Quality: 1}
}

// WeaponList 武器持有列表（供 HeroList 聚合下发与 GM 查询）。
func (this *weaponLogic) WeaponList(ctx context.Context, playerID int64) ([]*pb.WeaponEntry, error) {
	row, err := this.Load(ctx, playerID)
	if err != nil {
		return nil, err
	}

	entries := make([]*pb.WeaponEntry, 0, row.Weapons.Len())
	row.Weapons.Range(func(_ int32, w model.WeaponInfo) bool {
		entries = append(entries, convert.WeaponEntry(w))
		return true
	})
	// map 遍历无序，按配置ID排序保证客户端展示稳定
	sort.Slice(entries, func(i, j int) bool { return entries[i].CfgId < entries[j].CfgId })
	return entries, nil
}

// WeaponUpdate 更新武器：整体覆盖持有状态，未拥有返回错误。
func (this *weaponLogic) WeaponUpdate(ctx context.Context, playerID int64, req *pb.WeaponUpdateReq) (*pb.WeaponUpdateResp, error) {
	row, err := this.Load(ctx, playerID)
	if err != nil {
		return nil, err
	}

	w := model.WeaponInfo{
		WeaponID:     req.GetCfgId(),
		Level:        clampDim(req.GetLevel(), 1),
		Durability:   clampDim(req.GetDurability(), 0),
		Quality:      clampDim(req.GetQuality(), 1),
		Breakthrough: clampDim(req.GetBreakthrough(), 0),
	}
	if !row.UpdateWeapon(w) {
		return &pb.WeaponUpdateResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "武器未拥有")}, nil
	}

	this.Save(playerID, row)
	notifyHeroByWeapon(ctx, playerID, w.WeaponID)
	return &pb.WeaponUpdateResp{WeaponEntry: convert.WeaponEntry(w)}, nil
}

// weaponOwned 取玩家指定武器持有与其绑定英雄ID。
func weaponOwned(ctx context.Context, playerID int64, cfgID int32) (*model.PlayerWeapon, model.WeaponInfo, int32, error) {
	row, err := Weapon().Load(ctx, playerID)
	if err != nil {
		return nil, model.WeaponInfo{}, 0, err
	}
	w, ok := row.Weapons.Get(cfgID)
	if !ok {
		return nil, model.WeaponInfo{}, 0, errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "武器未拥有")
	}
	heroID, ok := heroIDByWeapon(cfgID)
	if !ok {
		return nil, model.WeaponInfo{}, 0, errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "武器未绑定英雄")
	}
	return row, w, heroID, nil
}

// weaponDimGrow 维度提升通用流程：校验持有 → 取当前维度行得消耗 → 校验下一维度行存在 → 扣素材 → 维度+1。
// rowOf 取指定维度值的等级/品质/突破行（素材与消耗字段由调用方读取）。
func weaponDimGrow(ctx context.Context, playerID int64, cfgID int32,
	cur func(w model.WeaponInfo) int32,
	set func(w *model.WeaponInfo, v int32),
	rowOf func(dim int32) (material, use int32, ok bool),
	fullMsg string,
) (*model.PlayerWeapon, model.WeaponInfo, int32, error) {
	row, w, heroID, err := weaponOwned(ctx, playerID, cfgID)
	if err != nil {
		return nil, model.WeaponInfo{}, 0, err
	}
	dim := cur(w)
	material, use, ok := rowOf(dim)
	if !ok {
		return nil, model.WeaponInfo{}, 0, errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "武器维度配置不存在")
	}
	_, _, hasNext := rowOf(dim + 1)
	if err := dimConsume(ctx, playerID, material, use, hasNext, fullMsg); err != nil {
		return nil, model.WeaponInfo{}, 0, err
	}
	set(&w, dim+1)
	if !row.UpdateWeapon(w) {
		return nil, model.WeaponInfo{}, 0, errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "武器未拥有")
	}
	weaponCache.Set(playerID, row)
	notifyHero(ctx, playerID, heroID)
	return row, w, heroID, nil
}

// WeaponUpgrade 武器升级：消耗 WeaponLevel 表当前等级的升级素材，等级 +1。
func (this *weaponLogic) WeaponUpgrade(ctx context.Context, playerID int64, req *pb.WeaponUpgradeReq) (*pb.WeaponUpgradeResp, error) {
	_, w, heroID, err := weaponDimGrow(ctx, playerID, req.GetCfgId(),
		func(w model.WeaponInfo) int32 { return w.Level },
		func(w *model.WeaponInfo, v int32) { w.Level = v },
		func(dim int32) (int32, int32, bool) {
			c := tpl.T().WeaponLevel.Get(dimID(req.GetCfgId(), dim))
			if c == nil {
				return 0, 0, false
			}
			return c.UpgradeMaterial, c.UpgradeUse, true
		},
		"武器已满级")
	if err != nil {
		if r := codedResult(err); r != nil {
			return &pb.WeaponUpgradeResp{Result: r}, nil
		}
		return nil, err
	}
	record.Record(ctx, model.RecordModuleWeapon, playerID, model.RecordActionConsume, req.GetCfgId(), 1)

	_, attr, err := computeHeroAttribute(ctx, playerID, heroID)
	if err != nil {
		if r := codedResult(err); r != nil {
			return &pb.WeaponUpgradeResp{Result: r}, nil
		}
		return nil, err
	}
	return &pb.WeaponUpgradeResp{WeaponEntry: convert.WeaponEntry(w), Attribute: attr}, nil
}

// WeaponUpQuality 武器升品：消耗 WeaponQuality 表当前品质的升品素材，品质 +1。
func (this *weaponLogic) WeaponUpQuality(ctx context.Context, playerID int64, req *pb.WeaponUpQualityReq) (*pb.WeaponUpQualityResp, error) {
	_, w, heroID, err := weaponDimGrow(ctx, playerID, req.GetCfgId(),
		func(w model.WeaponInfo) int32 { return w.Quality },
		func(w *model.WeaponInfo, v int32) { w.Quality = v },
		func(dim int32) (int32, int32, bool) {
			c := tpl.T().WeaponQuality.Get(dimID(req.GetCfgId(), dim))
			if c == nil {
				return 0, 0, false
			}
			return c.QualityMaterial, c.QualityUse, true
		},
		"武器已满品质")
	if err != nil {
		if r := codedResult(err); r != nil {
			return &pb.WeaponUpQualityResp{Result: r}, nil
		}
		return nil, err
	}
	record.Record(ctx, model.RecordModuleWeapon, playerID, model.RecordActionConsume, req.GetCfgId(), 1)

	_, attr, err := computeHeroAttribute(ctx, playerID, heroID)
	if err != nil {
		if r := codedResult(err); r != nil {
			return &pb.WeaponUpQualityResp{Result: r}, nil
		}
		return nil, err
	}
	return &pb.WeaponUpQualityResp{WeaponEntry: convert.WeaponEntry(w), Attribute: attr}, nil
}

// WeaponBreakthrough 武器突破：消耗 WeaponBreakthrough 表当前突破的素材，突破 +1。
func (this *weaponLogic) WeaponBreakthrough(ctx context.Context, playerID int64, req *pb.WeaponBreakthroughReq) (*pb.WeaponBreakthroughResp, error) {
	_, w, heroID, err := weaponDimGrow(ctx, playerID, req.GetCfgId(),
		func(w model.WeaponInfo) int32 { return w.Breakthrough },
		func(w *model.WeaponInfo, v int32) { w.Breakthrough = v },
		func(dim int32) (int32, int32, bool) {
			c := tpl.T().WeaponBreakthrough.Get(dimID(req.GetCfgId(), dim))
			if c == nil {
				return 0, 0, false
			}
			return c.BreakthroughMaterial, c.BreakthroughUse, true
		},
		"武器已满突破")
	if err != nil {
		if r := codedResult(err); r != nil {
			return &pb.WeaponBreakthroughResp{Result: r}, nil
		}
		return nil, err
	}
	record.Record(ctx, model.RecordModuleWeapon, playerID, model.RecordActionConsume, req.GetCfgId(), 1)

	_, attr, err := computeHeroAttribute(ctx, playerID, heroID)
	if err != nil {
		if r := codedResult(err); r != nil {
			return &pb.WeaponBreakthroughResp{Result: r}, nil
		}
		return nil, err
	}
	return &pb.WeaponBreakthroughResp{WeaponEntry: convert.WeaponEntry(w), Attribute: attr}, nil
}

// WeaponEnsureForHero 按英雄配置初始化固定绑定武器：未持有则补建默认状态。
func (this *weaponLogic) WeaponEnsureForHero(ctx context.Context, playerID int64, heroCfgID int32) error {
	cfg := tpl.T().RoleInfo.Get(heroCfgID)
	if cfg == nil || cfg.WeaponId <= 0 {
		return nil
	}
	row, err := this.Load(ctx, playerID)
	if err != nil {
		return err
	}
	if row.Weapons.Has(cfg.WeaponId) {
		return nil
	}
	if !row.AddWeapon(newWeaponInfo(cfg.WeaponId)) {
		return nil
	}
	this.Save(playerID, row)
	return nil
}

// EnterGame 进入游戏：加载玩家武器持有，未建档则初始化空集合；并按已拥有英雄兜底补建绑定武器。
func (this *weaponLogic) OnEnterGame(ctx context.Context, playerID int64, _ *pb.EnterGameReq, _ *pb.EnterGameResp) error {
	if _, err := this.Load(ctx, playerID); err != nil {
		return err
	}
	heroes, err := Hero().Load(ctx, playerID)
	if err != nil {
		return err
	}
	if heroes.Heroes != nil {
		var ids []int32
		heroes.Heroes.Range(func(id int32, _ model.HeroInfo) bool {
			ids = append(ids, id)
			return true
		})
		for _, id := range ids {
			if err := this.WeaponEnsureForHero(ctx, playerID, id); err != nil {
				return err
			}
		}
	}
	// 武器数据由 WeaponList 下发，这里只预热缓存
	return nil
}

// LeaveGame 离开游戏：武器持有数据同步落库，并逐出内存缓存。
func (this *weaponLogic) OnLeaveGame(ctx context.Context, playerID int64, _ *pb.LeaveGameReq, _ *pb.LeaveGameResp) error {
	if err := flushAndEvict(ctx, weaponCache, playerID); err != nil {
		return err
	}
	return nil
}

// Load 加载玩家武器持有；未建档则初始化空 map 并挂入缓存（异步落库）。
func (this *weaponLogic) Load(ctx context.Context, playerID int64) (*model.PlayerWeapon, error) {
	return weaponCache.GetOrInit(ctx, playerID,
		func(id int64) *model.PlayerWeapon {
			return &model.PlayerWeapon{PlayerID: id, Weapons: util.NewSafeMap[int32, model.WeaponInfo]()}
		},
		func(row *model.PlayerWeapon) {
			if row.Weapons == nil {
				row.Weapons = util.NewSafeMap[int32, model.WeaponInfo]()
			}
		})
}

// Save 写回玩家武器持有并挂入异步落库队列。
func (this *weaponLogic) Save(playerID int64, row *model.PlayerWeapon) {
	weaponCache.Set(playerID, row)
}

// Flush 同步落库并逐出缓存（GM 直改 DB 后强制回源）。
func (this *weaponLogic) Flush(ctx context.Context, playerID int64) error {
	return flushAndEvict(ctx, weaponCache, playerID)
}

// ============ 资产接口实现（AssetStore，唯一持有型：key 为武器配置ID） ============

// weaponAssetStore 武器资产实现：落在 player_weapon。武器由英雄配置绑定，发放幂等。
type weaponAssetStore struct{}

// AssetType 资产类型：武器。
func (this *weaponAssetStore) AssetType() tpl.AssetType { return tpl.AssetType_Weapon }

// Add 发放武器：已拥有视为成功（武器随英雄自动补建，避免奖励重复发放失败）。
func (this *weaponAssetStore) Add(ctx context.Context, playerID int64, cfgID, _ int32) error {
	if !tpl.T().Weapon.Has(cfgID) {
		return errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "武器配置不存在")
	}
	row, err := Weapon().Load(ctx, playerID)
	if err != nil {
		return err
	}
	if row.Weapons.Has(cfgID) {
		return nil
	}
	if !row.AddWeapon(newWeaponInfo(cfgID)) {
		return errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "武器已拥有")
	}

	weaponCache.Set(playerID, row)
	record.Record(ctx, model.RecordModuleWeapon, playerID, model.RecordActionGain, cfgID, 1)
	notifyHeroByWeapon(ctx, playerID, cfgID)
	return nil
}

// Remove 扣除武器；未拥有返回错误。
func (this *weaponAssetStore) Remove(ctx context.Context, playerID int64, cfgID, _ int32) error {
	row, err := Weapon().Load(ctx, playerID)
	if err != nil {
		return err
	}
	if !row.Weapons.Delete(cfgID) {
		return errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "武器未拥有")
	}

	weaponCache.Set(playerID, row)
	record.Record(ctx, model.RecordModuleWeapon, playerID, model.RecordActionConsume, cfgID, 1)
	notifyHeroByWeapon(ctx, playerID, cfgID)
	return nil
}

// Update 覆盖持有：count<=0 移除，>0 确保持有。
func (this *weaponAssetStore) Update(ctx context.Context, playerID int64, cfgID, count int32) error {
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
func (this *weaponAssetStore) Query(ctx context.Context, playerID int64, cfgID int32) (int32, error) {
	row, err := Weapon().Load(ctx, playerID)
	if err != nil {
		return 0, err
	}
	if row.Weapons.Has(cfgID) {
		return 1, nil
	}
	return 0, nil
}

// List 列出全部武器（按配置ID升序）。
func (this *weaponAssetStore) List(ctx context.Context, playerID int64) ([]*tpl.Asset, error) {
	row, err := Weapon().Load(ctx, playerID)
	if err != nil {
		return nil, err
	}

	assets := make([]*tpl.Asset, 0, row.Weapons.Len())
	row.Weapons.Range(func(cfgID int32, _ model.WeaponInfo) bool {
		assets = append(assets, &tpl.Asset{Type: tpl.AssetType_Weapon, Id: cfgID, Count: 1})
		return true
	})
	sort.Slice(assets, func(i, j int) bool { return assets[i].Id < assets[j].Id })
	return assets, nil
}

// AttributeBonus 武器模块对英雄的伤害贡献：基础伤害 × 等级系数 × 品质系数 × 突破系数。
// 基础伤害取 AttributeWeapon 的 1 星档行；英雄未绑定武器、玩家未持有或配置缺失时为 0。
func (this *weaponLogic) AttributeBonus(ctx context.Context, playerID int64, heroID int32) (*pb.HeroAttribute, error) {
	cfg := tpl.T().RoleInfo.Get(heroID)
	if cfg == nil || cfg.WeaponId <= 0 {
		return &pb.HeroAttribute{}, nil
	}
	row, err := this.Load(ctx, playerID)
	if err != nil {
		return nil, err
	}
	w, ok := row.Weapons.Get(cfg.WeaponId)
	if !ok {
		return &pb.HeroAttribute{}, nil
	}
	base := tpl.T().AttributeWeapon.Get(attrID(1, cfg.WeaponId))
	if base == nil {
		return &pb.HeroAttribute{}, nil
	}
	coef := coefOr1(weaponLevelCoef(cfg.WeaponId, w.Level)) *
		coefOr1(weaponQualityCoef(cfg.WeaponId, w.Quality)) *
		coefOr1(weaponBreakthroughCoef(cfg.WeaponId, w.Breakthrough))
	return &pb.HeroAttribute{Damage: int32(float64(base.WeaponDamage) * coef)}, nil
}

// 武器维度系数查询（行缺失按 1）。
func weaponLevelCoef(cfgID, level int32) float32 {
	if c := tpl.T().WeaponLevel.Get(dimID(cfgID, level)); c != nil {
		return c.DamageCoef
	}
	return 1
}

func weaponQualityCoef(cfgID, quality int32) float32 {
	if c := tpl.T().WeaponQuality.Get(dimID(cfgID, quality)); c != nil {
		return c.DamageCoef
	}
	return 1
}

func weaponBreakthroughCoef(cfgID, breakthrough int32) float32 {
	if c := tpl.T().WeaponBreakthrough.Get(dimID(cfgID, breakthrough)); c != nil {
		return c.DamageCoef
	}
	return 1
}
