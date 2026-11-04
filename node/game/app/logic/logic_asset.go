package logic

import (
	"context"
	"fmt"

	"server-cluster/node/game/app/global"
	"server-cluster/share/convert"
	"server-cluster/share/errs"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
	"server-cluster/share/tpl"
)

type assetLogic struct{}

var assetLogicInst = singleton.NewContainer[global.AssetLogic]("AssetLogic")

// assetHandlers 资产类型 → 单类实现（item/equip/weapon/skill/hero 在各自 init 中注册）。
var assetHandlers = map[tpl.AssetType]global.AssetStore{}

func init() {
	assetLogicInst.Register(&assetLogic{})
}

// Asset 获取资产业务单例。
func Asset() global.AssetLogic { return assetLogicInst.MustGet() }

// registerAsset 注册单类资产实现。重复注册直接 panic，启动期暴露配置错误。
func registerAsset(h global.AssetStore) {
	t := h.AssetType()
	if _, dup := assetHandlers[t]; dup {
		panic(fmt.Sprintf("logic: duplicate asset handler for type %d", t))
	}
	assetHandlers[t] = h
}

// assetHandler 按类型取实现。
func assetHandler(t tpl.AssetType) (global.AssetStore, error) {
	h, ok := assetHandlers[t]
	if !ok {
		return nil, errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, fmt.Sprintf("不支持的资产类型: %d", t))
	}
	return h, nil
}

// apply 分发增删改：校验资产非空、按类型取实现后交给具体操作。
func (this *assetLogic) apply(ctx context.Context, playerID int64, asset *tpl.Asset,
	op func(h global.AssetStore, ctx context.Context, playerID int64, cfgID, count int32) error) error {
	if asset == nil {
		return errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "资产为空")
	}
	h, err := assetHandler(asset.Type)
	if err != nil {
		return err
	}
	return op(h, ctx, playerID, asset.Id, asset.Count)
}

// applyAsset 分发带返回值的查询：按类型取实现后交给具体操作。
func applyAsset[R any](t tpl.AssetType, op func(h global.AssetStore) (R, error)) (R, error) {
	var zero R
	h, err := assetHandler(t)
	if err != nil {
		return zero, err
	}
	return op(h)
}

func (this *assetLogic) AssetAdd(ctx context.Context, playerID int64, asset *tpl.Asset) error {
	return this.apply(ctx, playerID, asset, global.AssetStore.Add)
}

func (this *assetLogic) AssetRemove(ctx context.Context, playerID int64, asset *tpl.Asset) error {
	return this.apply(ctx, playerID, asset, global.AssetStore.Remove)
}

func (this *assetLogic) AssetUpdate(ctx context.Context, playerID int64, asset *tpl.Asset) error {
	return this.apply(ctx, playerID, asset, global.AssetStore.Update)
}

func (this *assetLogic) AssetQuery(ctx context.Context, playerID int64, assetType tpl.AssetType, cfgID int32) (int32, error) {
	return applyAsset(assetType, func(h global.AssetStore) (int32, error) {
		return h.Query(ctx, playerID, cfgID)
	})
}

func (this *assetLogic) AssetList(ctx context.Context, playerID int64, assetType tpl.AssetType) ([]*tpl.Asset, error) {
	return applyAsset(assetType, func(h global.AssetStore) ([]*tpl.Asset, error) {
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
		entries = append(entries, convert.AssetEntry(a))
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
		entries = append(entries, convert.AssetEntry(a))
	}
	return entries, nil
}
