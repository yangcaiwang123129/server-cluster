package logic

import (
	"context"
	"fmt"

	"server-cluster/node/game/app/global"
	"server-cluster/share/errs"
	"server-cluster/share/singleton"
	"server-cluster/share/tpl"
)

// AssetTypeSkill 技能资产类型。
// tpl.AssetType 生成自 cfg/excel/__enums__.xlsx，目前只到 Weapon(4)；技能先占用 5，
// 待枚举表补上“技能”并重新生成配置后，把这里换成 tpl.AssetType_Skill 即可。
const AssetTypeSkill tpl.AssetType = 5

type assetLogic struct{}

var assetLogicInst = singleton.NewContainer[global.AssetLogic]("AssetLogic")

// assetHandlers 资产类型 → 单类实现（item/equip/weapon/skill/hero 在各自 init 中注册）。
var assetHandlers = map[tpl.AssetType]global.AssetHandler{}

func init() {
	assetLogicInst.Register(&assetLogic{})
}

// Asset 获取资产业务单例。
func Asset() global.AssetLogic { return assetLogicInst.MustGet() }

// registerAssetHandler 注册单类资产实现；重复注册直接 panic，启动期暴露配置错误。
func registerAssetHandler(h global.AssetHandler) {
	t := h.AssetType()
	if _, dup := assetHandlers[t]; dup {
		panic(fmt.Sprintf("logic: duplicate asset handler for type %d", t))
	}
	assetHandlers[t] = h
}

// assetHandler 按类型取实现。
func assetHandler(t tpl.AssetType) (global.AssetHandler, error) {
	h, ok := assetHandlers[t]
	if !ok {
		return nil, errs.CodeInvalidParams.WithMessage(fmt.Sprintf("不支持的资产类型: %d", t)).Err()
	}
	return h, nil
}

// AssetAdd 发放资产。
func (this *assetLogic) AssetAdd(ctx context.Context, playerID int64, asset *tpl.Asset) error {
	if asset == nil {
		return errs.CodeInvalidParams.WithMessage("资产为空").Err()
	}
	h, err := assetHandler(asset.Type)
	if err != nil {
		return err
	}
	return h.Add(ctx, playerID, asset.Id, asset.Count)
}

// AssetRemove 扣除资产。
func (this *assetLogic) AssetRemove(ctx context.Context, playerID int64, asset *tpl.Asset) error {
	if asset == nil {
		return errs.CodeInvalidParams.WithMessage("资产为空").Err()
	}
	h, err := assetHandler(asset.Type)
	if err != nil {
		return err
	}
	return h.Remove(ctx, playerID, asset.Id, asset.Count)
}

// AssetUpdate 覆盖持有状态。
func (this *assetLogic) AssetUpdate(ctx context.Context, playerID int64, asset *tpl.Asset) error {
	if asset == nil {
		return errs.CodeInvalidParams.WithMessage("资产为空").Err()
	}
	h, err := assetHandler(asset.Type)
	if err != nil {
		return err
	}
	return h.Update(ctx, playerID, asset.Id, asset.Count)
}

// AssetQuery 查询持有数量。
func (this *assetLogic) AssetQuery(ctx context.Context, playerID int64, assetType tpl.AssetType, cfgID int32) (int32, error) {
	h, err := assetHandler(assetType)
	if err != nil {
		return 0, err
	}
	return h.Query(ctx, playerID, cfgID)
}

// AssetList 列出某类资产全部持有。
func (this *assetLogic) AssetList(ctx context.Context, playerID int64, assetType tpl.AssetType) ([]*tpl.Asset, error) {
	h, err := assetHandler(assetType)
	if err != nil {
		return nil, err
	}
	return h.List(ctx, playerID)
}
