package logic

import (
	"context"
	"server-cluster/share/consts"
	"sort"

	"github.com/dobyte/due/v2/log"
	"github.com/dobyte/due/v2/utils/xconv"

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

type heroLogic struct {
	cache *cache.Cache[int64, *model.PlayerHero]
}

var heroLogicInst = singleton.NewContainer[global.HeroLogic]("HeroLogic")

func init() {
	hLogic := &heroLogic{cache: cache.NewPlayerTableCache[model.PlayerHero](cache.CacherConfig{Name: enum.TablePlayerHero.String()})}
	heroLogicInst.Register(hLogic)
	cache.RegisterCacheCloser(hLogic.cache)
	registerAsset(&heroAssetStore{})
}

// Hero 获取英雄业务单例。
func Hero() global.HeroLogic { return heroLogicInst.MustGet() }

func (this *heroLogic) OnEnterGame(ctx context.Context, playerID int64, _ *pb.EnterGameReq, resp *pb.EnterGameResp) error {
	entries, err := buildHeroEntries(ctx, playerID)
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

func (this *heroLogic) OnLeaveGame(ctx context.Context, playerID int64, _ *pb.LeaveGameReq, _ *pb.LeaveGameResp) error {
	if err := flushAndEvict(ctx, this.cache, playerID); err != nil {
		return err
	}
	return nil
}

func (this *heroLogic) Load(ctx context.Context, playerID int64) (*model.PlayerHero, error) {
	return this.cache.GetOrInit(ctx, playerID,
		func(id int64) *model.PlayerHero {
			playerHero := &model.PlayerHero{PlayerID: id, Heroes: util.NewSafeMap[int32, model.HeroInfo]()}
			value := tpl.T().Global.Get(consts.CreatePlayerGiveHero).Value
			cfgID := xconv.Int32(value)
			if err := grantHeroBound(ctx, playerID, int64(cfgID)); err != nil {
				return playerHero
			}

			heroInfo := newHeroInfo(cfgID)
			heroInfo.Equipped = true
			playerHero.AddHero(heroInfo)
			record.Record(ctx, model.RecordModuleHero, playerID, model.RecordActionGain, cfgID, 1)
			return playerHero
		},
		func(row *model.PlayerHero) {
			if row.Heroes == nil {
				row.Heroes = util.NewSafeMap[int32, model.HeroInfo]()
			}
		})
}

func (this *heroLogic) Save(playerID int64, row *model.PlayerHero) {
	this.cache.Set(playerID, row)
}

func (this *heroLogic) Flush(ctx context.Context, playerID int64) error {
	return flushAndEvict(ctx, this.cache, playerID)
}
func (this *heroLogic) HeroList(ctx context.Context, playerID int64) (*pb.HeroListResp, error) {
	entries, err := buildHeroEntries(ctx, playerID)
	if err != nil {
		return nil, err
	}
	return &pb.HeroListResp{HeroEntries: entries}, nil
}

// buildHeroEntries 构建玩家完整英雄数据列表
func buildHeroEntries(ctx context.Context, playerID int64) ([]*pb.HeroEntry, error) {
	row, err := Hero().Load(ctx, playerID)
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

// buildHeroEntry 构建玩家单英雄数据
func buildHeroEntry(ctx context.Context, playerID int64, heroID int32) (*pb.HeroEntry, error) {
	entries, err := buildHeroEntries(ctx, playerID)
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

// notifyHeroByID 按英雄ID组装完整持有结构后推送；英雄未拥有时只记日志不推送。
func notifyHeroByID(ctx context.Context, playerID int64, heroID int32) {
	entry, err := buildHeroEntry(ctx, playerID, heroID)
	if err != nil {
		log.Warnf("game: assemble hero entry failed, player=%d hero=%d: %v", playerID, heroID, err)
		return
	}
	global.NotifyHero(ctx, playerID, entry)
}

// notifyHeroByWeapon 武器变化后按固定绑定反查所属英雄并推送。
func notifyHeroByWeapon(ctx context.Context, playerID int64, cfgID int32) {
	heroID, ok := heroIDByWeapon(cfgID)
	if !ok {
		return
	}
	notifyHeroByID(ctx, playerID, heroID)
}

// notifyHeroBySkill 技能变化后按固定绑定反查所属英雄并推送。
func notifyHeroBySkill(ctx context.Context, playerID int64, cfgID int32) {
	heroID, ok := heroIDBySkill(cfgID)
	if !ok {
		return
	}
	notifyHeroByID(ctx, playerID, heroID)
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
func addHeroWithBound(ctx context.Context, playerID int64, cfgID int32) (model.HeroInfo, error) {
	row, err := Hero().Load(ctx, playerID)
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
	Hero().(*heroLogic).cache.Set(playerID, row)
	record.Record(ctx, model.RecordModuleHero, playerID, model.RecordActionGain, cfgID, 1)
	return h, nil
}

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
	entry, err := buildHeroEntry(ctx, playerID, h.HeroID)
	if err != nil {
		if r := codedResult(err); r != nil {
			return &pb.HeroAddResp{Result: r}, nil
		}
		return nil, err
	}
	global.NotifyHero(ctx, playerID, entry)
	return &pb.HeroAddResp{HeroEntry: entry}, nil
}

func (this *heroLogic) HeroUpdate(ctx context.Context, playerID int64, req *pb.HeroUpdateReq) (*pb.HeroUpdateResp, error) {
	row, err := this.Load(ctx, playerID)
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
	entry, err := buildHeroEntry(ctx, playerID, h.HeroID)
	if err != nil {
		if r := codedResult(err); r != nil {
			return &pb.HeroUpdateResp{Result: r}, nil
		}
		return nil, err
	}
	for _, id := range unsetHeroIDs {
		notifyHeroByID(ctx, playerID, id)
	}
	global.NotifyHero(ctx, playerID, entry)
	return &pb.HeroUpdateResp{HeroEntry: entry}, nil
}

func (this *heroLogic) HeroAttribute(ctx context.Context, playerID int64, req *pb.HeroAttributeReq) (*pb.HeroAttributeResp, error) {
	hero, attr, err := computeHeroAttribute(ctx, playerID, req.GetCfgId())
	if err != nil {
		if r := codedResult(err); r != nil {
			return &pb.HeroAttributeResp{Result: r}, nil
		}
		return nil, err
	}
	entry, err := buildHeroEntry(ctx, playerID, hero.HeroID)
	if err != nil {
		if r := codedResult(err); r != nil {
			return &pb.HeroAttributeResp{Result: r}, nil
		}
		return nil, err
	}
	return &pb.HeroAttributeResp{HeroEntry: entry, Attribute: attr}, nil
}

func (this *heroLogic) HeroUpgrade(ctx context.Context, playerID int64, req *pb.HeroUpgradeReq) (*pb.HeroUpgradeResp, error) {
	row, err := this.Load(ctx, playerID)
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
	entry, err := buildHeroEntry(ctx, playerID, hero.HeroID)
	if err != nil {
		if r := codedResult(err); r != nil {
			return &pb.HeroUpgradeResp{Result: r}, nil
		}
		return nil, err
	}
	global.NotifyHero(ctx, playerID, entry)
	return &pb.HeroUpgradeResp{HeroEntry: entry, Attribute: attr}, nil
}

// ============ 资产接口实现（AssetStore，唯一持有型） ============

// heroAssetStore 英雄资产实现：落在 player_hero。
type heroAssetStore struct{}

func (this *heroAssetStore) AssetType() tpl.AssetType { return tpl.AssetType_Hero }

func (this *heroAssetStore) Add(ctx context.Context, playerID int64, cfgID, _ int32) error {
	if !tpl.T().RoleInfo.Has(cfgID) {
		return errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "英雄配置不存在")
	}
	h, err := addHeroWithBound(ctx, playerID, cfgID)
	if err != nil {
		return err
	}
	notifyHeroByID(ctx, playerID, h.HeroID)
	return nil
}

func (this *heroAssetStore) Remove(ctx context.Context, playerID int64, cfgID, _ int32) error {
	row, err := Hero().Load(ctx, playerID)
	if err != nil {
		return err
	}
	if !row.Heroes.Delete(cfgID) {
		return errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "英雄未拥有")
	}

	Hero().(*heroLogic).cache.Set(playerID, row)
	record.Record(ctx, model.RecordModuleHero, playerID, model.RecordActionConsume, cfgID, 1)
	return nil
}

func (this *heroAssetStore) Update(ctx context.Context, playerID int64, cfgID, count int32) error {
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

func (this *heroAssetStore) Query(ctx context.Context, playerID int64, cfgID int32) (int32, error) {
	row, err := Hero().Load(ctx, playerID)
	if err != nil {
		return 0, err
	}
	if _, ok := row.Heroes.Get(cfgID); !ok {
		return 0, nil
	}
	return 1, nil
}

func (this *heroAssetStore) List(ctx context.Context, playerID int64) ([]*tpl.Asset, error) {
	row, err := Hero().Load(ctx, playerID)
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

func (this *heroLogic) AttributeBonus(ctx context.Context, playerID int64, heroID int32) (*pb.HeroAttribute, error) {
	row, err := this.Load(ctx, playerID)
	if err != nil {
		return nil, err
	}
	h, ok := row.Heroes.Get(heroID)
	if !ok {
		return &pb.HeroAttribute{}, nil
	}
	return heroBaseAttribute(h)
}
