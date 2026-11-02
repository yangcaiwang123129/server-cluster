package logic

import (
	"context"
	"fmt"

	"server-cluster/node/game/app/global"
	"server-cluster/share/errs"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
	"server-cluster/share/tpl"
)

// AssetTypeSkill 技能资产类型。
const AssetType_Weapon tpl.AssetType = 4
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

// registerAsset 注册单类资产实现。重复注册直接 panic，启动期暴露配置错误。
func registerAsset(h global.AssetHandler) {
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

// apply 分发增删改：校验资产非空、按类型取实现后交给具体操作。
func (this *assetLogic) apply(ctx context.Context, playerID int64, asset *tpl.Asset,
	op func(h global.AssetHandler, ctx context.Context, playerID int64, cfgID, count int32) error) error {
	if asset == nil {
		return errs.CodeInvalidParams.WithMessage("资产为空").Err()
	}
	h, err := assetHandler(asset.Type)
	if err != nil {
		return err
	}
	return op(h, ctx, playerID, asset.Id, asset.Count)
}

// applyAsset 分发带返回值的查询：按类型取实现后交给具体操作。
func applyAsset[R any](t tpl.AssetType, op func(h global.AssetHandler) (R, error)) (R, error) {
	var zero R
	h, err := assetHandler(t)
	if err != nil {
		return zero, err
	}
	return op(h)
}

// AssetAdd 发放资产。
func (this *assetLogic) AssetAdd(ctx context.Context, playerID int64, asset *tpl.Asset) error {
	return this.apply(ctx, playerID, asset, global.AssetHandler.Add)
}

// AssetRemove 扣除资产。
func (this *assetLogic) AssetRemove(ctx context.Context, playerID int64, asset *tpl.Asset) error {
	return this.apply(ctx, playerID, asset, global.AssetHandler.Remove)
}

// AssetUpdate 覆盖持有状态。
func (this *assetLogic) AssetUpdate(ctx context.Context, playerID int64, asset *tpl.Asset) error {
	return this.apply(ctx, playerID, asset, global.AssetHandler.Update)
}

// AssetQuery 查询持有数量。
func (this *assetLogic) AssetQuery(ctx context.Context, playerID int64, assetType tpl.AssetType, cfgID int32) (int32, error) {
	return applyAsset(assetType, func(h global.AssetHandler) (int32, error) {
		return h.Query(ctx, playerID, cfgID)
	})
}

// AssetList 列出某类资产全部持有。
func (this *assetLogic) AssetList(ctx context.Context, playerID int64, assetType tpl.AssetType) ([]*tpl.Asset, error) {
	return applyAsset(assetType, func(h global.AssetHandler) ([]*tpl.Asset, error) {
		return h.List(ctx, playerID)
	})
}

// deductAssets 依次扣除资产并汇总 wire 条目；nil 条目跳过，任一失败即返回错误（不继续后续扣减）。
func deductAssets(ctx context.Context, playerID int64, assets []*tpl.Asset) ([]*pb.AssetEntry, error) {
	entries := make([]*pb.AssetEntry, 0, len(assets))
	for _, a := range assets {
		if a == nil {
			continue
		}
		if err := Asset().AssetRemove(ctx, playerID, a); err != nil {
			return nil, err
		}
		entries = append(entries, toAssetEntry(a))
	}
	return entries, nil
}

// grantAssets 依次发放资产并汇总 wire 条目；nil 条目跳过，任一失败即返回错误。
func grantAssets(ctx context.Context, playerID int64, assets []*tpl.Asset) ([]*pb.AssetEntry, error) {
	entries := make([]*pb.AssetEntry, 0, len(assets))
	for _, a := range assets {
		if a == nil {
			continue
		}
		if err := Asset().AssetAdd(ctx, playerID, a); err != nil {
			return nil, err
		}
		entries = append(entries, toAssetEntry(a))
	}
	return entries, nil
}

// toAssetEntry 资产配置 → wire 条目；Count<=0 与 AssetAdd 一致按 1 处理。
func toAssetEntry(a *tpl.Asset) *pb.AssetEntry {
	count := a.Count
	if count <= 0 {
		count = 1
	}
	return &pb.AssetEntry{Type: int32(a.Type), CfgId: a.Id, Count: count}
}
