// logic_currency.go 货币类资产（点券/金币/钻石）：不走 playerCache 的异步批量落库，
// 增减直接执行 DB 原子 SQL（余额守卫防超扣），事务成功后把结果同步进已缓存的内存行。
//
// 设计要点：
//  1. 绕过缓存异步落库——货币是强一致敏感数据，异步 upsert 的整行覆盖在并发下可能丢更新；
//  2. 原子 SQL——`SET col = col + ?` + `WHERE col + ? >= 0` 守卫，由数据库保证不超扣；
//  3. 内存行只同步不回写——事务成功后仅当玩家行已在 playerCache 中时改写对应字段
//     （Peek 只读探测，不 Set），避免全行 upsert 覆盖其他列的窗口期，DB 始终是事实源。
//
// 前提：playerCache 当前从不 Set（仅 OnLeaveGame flushAndEvict），无 pending 写路径，
// 本文件是该行货币字段唯一的内存写者；若未来引入 playerCache.Set 写路径，
// 需先消除与本同步逻辑的竞态。
package logic

import (
	"context"

	"gorm.io/gorm"

	"server-cluster/internal/svc"
	"server-cluster/node/game/app/global"
	"server-cluster/share/errs"
	"server-cluster/share/model"
	"server-cluster/share/pb"
	"server-cluster/share/tpl"
	"server-cluster/share/util"
)

// 货币列白名单：仅允许 player 表这三个货币列，SQL 拼接前逐一匹配，杜绝列名注入。
const (
	currencyColumnCoupon  = "coupon"  // 点券（model.Player.Coupon）
	currencyColumnGold    = "gold"    // 金币（model.Player.Gold）
	currencyColumnDiamond = "diamond" // 钻石（model.Player 字段名 Coin，列名 diamond）
)

// currencyAssetStore 货币资产实现：player 单行单列，原子 SQL 增减。
type currencyAssetStore struct {
	assetType tpl.AssetType
	column    string // 仅限上方白名单常量
}

func init() {
	registerAsset(&currencyAssetStore{assetType: tpl.AssetType_Coupon, column: currencyColumnCoupon})
	registerAsset(&currencyAssetStore{assetType: tpl.AssetType_Gold, column: currencyColumnGold})
	registerAsset(&currencyAssetStore{assetType: tpl.AssetType_Coin, column: currencyColumnDiamond})
}

func (this *currencyAssetStore) AssetType() tpl.AssetType { return this.assetType }

// Add 发放货币：货币无配置，cfgID 必须为 0；count<=0 按 1。
func (this *currencyAssetStore) Add(ctx context.Context, playerID int64, cfgID, count int32) error {
	if err := this.checkCfgID(cfgID); err != nil {
		return err
	}
	if count <= 0 {
		count = 1
	}
	return adjustCurrency(ctx, playerID, this.column, int64(count))
}

// Remove 扣除货币：count<=0 按 1；余额不足返回错误。
func (this *currencyAssetStore) Remove(ctx context.Context, playerID int64, cfgID, count int32) error {
	if err := this.checkCfgID(cfgID); err != nil {
		return err
	}
	if count <= 0 {
		count = 1
	}
	return adjustCurrency(ctx, playerID, this.column, -int64(count))
}

// Update 覆盖货币为指定值：count<=0 置 0，>0 直接 SET。
func (this *currencyAssetStore) Update(ctx context.Context, playerID int64, cfgID, count int32) error {
	if err := this.checkCfgID(cfgID); err != nil {
		return err
	}
	if count <= 0 {
		count = 0
	}
	return setCurrency(ctx, playerID, this.column, int64(count))
}

// Query 直查 DB 当前持有（保真，绕过内存缓存），行不存在返回 0。
func (this *currencyAssetStore) Query(ctx context.Context, playerID int64, cfgID int32) (int32, error) {
	db, err := svc.Ctx().DB()
	if err != nil {
		return 0, err
	}
	var value int64
	// 行不存在时 Scan 不报错且 value 保持 0，符合“行不存在返回 0”语义
	res := db.WithContext(ctx).Table(model.Player{}.TableName()).
		Select(this.column).
		Where("player_id = ?", playerID).
		Scan(&value)
	if res.Error != nil {
		return 0, res.Error
	}
	return int32(value), nil
}

// List 货币为单值资产：返回一条当前持有。
func (this *currencyAssetStore) List(ctx context.Context, playerID int64) ([]*tpl.Asset, error) {
	owned, err := this.Query(ctx, playerID, 0)
	if err != nil {
		return nil, err
	}
	return []*tpl.Asset{{Type: this.assetType, Count: owned}}, nil
}

// checkCfgID 货币资产无配置维度，cfgID 必须为 0。
func (this *currencyAssetStore) checkCfgID(cfgID int32) error {
	if cfgID != 0 {
		return errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "货币资产不使用配置ID")
	}
	return nil
}

// adjustCurrency 原子增减货币（delta 正加负减）：
// 事务内 `UPDATE player SET col = col + ? WHERE ...`；负 delta 附加余额守卫
// `col + ? >= 0`，守卫不满足时 0 行更新 → 返回余额不足错误回滚；
// 0 行更新且 delta 为正视为玩家行缺失。成功后同步内存行并推送货币快照。
func adjustCurrency(ctx context.Context, playerID int64, column string, delta int64) error {
	db, err := svc.Ctx().DB()
	if err != nil {
		return err
	}
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		q := tx.Table(model.Player{}.TableName()).Where("player_id = ?", playerID)
		if delta < 0 {
			// 余额守卫：col + delta >= 0，由数据库防并发超扣
			q = q.Where(column+" + ? >= 0", delta)
		}
		res := q.Updates(map[string]any{
			column:       gorm.Expr(column+" + ?", delta),
			"updated_at": util.NowUnixMilli(),
		})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			if delta < 0 {
				return errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "货币数量不足")
			}
			return errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "玩家不存在")
		}
		return nil
	})
	if err != nil {
		return err
	}
	syncCurrencyMemoryAndNotify(ctx, playerID, column, func(v int64) int64 { return v + delta })
	return nil
}

// setCurrency 覆盖货币为指定值（value >= 0，由调用方钳制）：成功后同步内存行并推送。
func setCurrency(ctx context.Context, playerID int64, column string, value int64) error {
	db, err := svc.Ctx().DB()
	if err != nil {
		return err
	}
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Table(model.Player{}.TableName()).
			Where("player_id = ?", playerID).
			Updates(map[string]any{
				column:       value,
				"updated_at": util.NowUnixMilli(),
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "玩家不存在")
		}
		return nil
	})
	if err != nil {
		return err
	}
	syncCurrencyMemoryAndNotify(ctx, playerID, column, func(int64) int64 { return value })
	return nil
}

// syncCurrencyMemoryAndNotify 事务成功后的内存同步：
// 仅当玩家行已在 playerCache 中（Peek 命中）时按白名单改写对应货币字段——
// 不 Set（避免整行 upsert 覆盖其他列的窗口期）、不回写 DB（DB 已落定），并推送三货币快照；
// 未命中（离线/未加载）则什么都不做。
func syncCurrencyMemoryAndNotify(ctx context.Context, playerID int64, column string, mutate func(int64) int64) {
	row, ok := playerCache.Peek(playerID)
	if !ok {
		return
	}
	switch column {
	case currencyColumnCoupon:
		row.Coupon = mutate(row.Coupon)
	case currencyColumnGold:
		row.Gold = mutate(row.Gold)
	case currencyColumnDiamond:
		row.Coin = mutate(row.Coin)
	default:
		return // 不在白名单，防御性跳过
	}
	global.NotifyAttr(ctx, playerID, &pb.AttrMap{Values: map[int32]*pb.ObjectValue{
		int32(pb.Attr_PLAYER_ATTR_COUPON): {Kind: &pb.ObjectValue_BigintValue{BigintValue: row.Coupon}},
		int32(pb.Attr_PLAYER_ATTR_GOLD):   {Kind: &pb.ObjectValue_BigintValue{BigintValue: row.Gold}},
		int32(pb.Attr_PLAYER_ATTR_COIN):   {Kind: &pb.ObjectValue_BigintValue{BigintValue: row.Coin}},
	}})
}
