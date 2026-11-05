// Package repo game 服数据访问层。
//
// repo_player.go 玩家账号行数据访问：player 表单行（账号信息 + 三货币列），
// 不做内存缓存，业务直查/直改 DB。
//
// 设计要点：
//  1. 货币原子 SQL——`SET col = col + ?`，负 delta 附加 `WHERE col + ? >= 0` 守卫，由数据库保证不超扣；
//  2. 列白名单——货币物品配置ID（enum.ItemID）映射到 player 表三个货币列（Coin→diamond、Gold→gold、Coupon→coupon），
//     未知 cfgID 报参数错误，杜绝列名注入；
//  3. 无状态包级函数——repo 只做 DB 访问，不持有状态，直接调用无需单例；
//     内存同步与客户端推送由 logic 层在事务成功后负责。
package repo

import (
	"context"
	"sync"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"server-cluster/internal/svc"
	"server-cluster/share/enum"
	"server-cluster/share/errs"
	"server-cluster/share/model"
	"server-cluster/share/pb"
	"server-cluster/share/util"
)

// QueryPlayer 按 ID 查询玩家账号行（含三货币列）；行不存在返回 gorm.ErrRecordNotFound。
func QueryPlayer(ctx context.Context, playerID int64) (*model.Player, error) {
	db, err := svc.Ctx().DB()
	if err != nil {
		return nil, gorm.ErrInvalidDB
	}
	var row model.Player
	if err := db.WithContext(ctx).First(&row, "player_id = ?", playerID).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// AdjustCurrency 原子增减货币：负 delta 附加余额守卫（0 行更新=货币数量不足），正 delta 0 行更新=玩家不存在。
func AdjustCurrency(ctx context.Context, playerID int64, cfgID int32, delta int64) error {
	column, ok := currencyColumn(cfgID)
	if !ok {
		return errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "未知货币类型")
	}
	db, err := svc.Ctx().DB()
	if err != nil {
		return gorm.ErrInvalidDB
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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
}

// SetCurrency 覆盖货币为指定值（value 由调用方保证非负，0 行更新=玩家不存在）。
func SetCurrency(ctx context.Context, playerID int64, cfgID int32, value int64) error {
	column, ok := currencyColumn(cfgID)
	if !ok {
		return errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "未知货币类型")
	}
	db, err := svc.Ctx().DB()
	if err != nil {
		return gorm.ErrInvalidDB
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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
}

// QueryCurrency 查询货币当前值；玩家行不存在返回 0。
func QueryCurrency(ctx context.Context, playerID int64, cfgID int32) (int64, error) {
	column, ok := currencyColumn(cfgID)
	if !ok {
		return 0, errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "未知货币类型")
	}
	db, err := svc.Ctx().DB()
	if err != nil {
		return 0, gorm.ErrInvalidDB
	}
	var value int64
	// 行不存在时 Scan 不报错且 value 保持 0，符合“行不存在返回 0”语义
	res := db.WithContext(ctx).Table(model.Player{}.TableName()).
		Select(column).
		Where("player_id = ?", playerID).
		Scan(&value)
	if res.Error != nil {
		return 0, res.Error
	}
	return value, nil
}

// playerColumns model.Player 的 Go 字段名 → DB 列名（gorm schema 启动时解析一次，embedded TableModel 自动展开），避免列名字符串硬编码。
var playerColumns = func() map[string]string {
	s, err := schema.Parse(&model.Player{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		panic(err)
	}
	m := make(map[string]string, len(s.Fields))
	for _, f := range s.Fields {
		m[f.Name] = f.DBName
	}
	return m
}()

// currencyColumn 货币物品配置ID → player 货币列（字段名经 enum.CurrencyFields 白名单，列名经 playerColumns 解析）；未知配置ID返回 false。
func currencyColumn(cfgID int32) (string, bool) {
	name, ok := enum.CurrencyFields[enum.ItemID(cfgID)]
	if !ok {
		return "", false
	}
	column, ok := playerColumns[name]
	if !ok || column == "" {
		return "", false
	}
	return column, true
}
