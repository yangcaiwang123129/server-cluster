package cache

import (
	"context"
	"errors"
	"reflect"
	"server-cluster/internal/svc"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// PlayerTableModel 玩家表模型约束：表名自述（gorm TableName），主键含 player_id。
type PlayerTableModel interface{ TableName() string }

// schemaCache 表结构解析缓存。
var schemaCache sync.Map

// upsertChunkSize 单条 upsert 语句最大行数（防占位符超限）。
const upsertChunkSize = 500

// tableMeta 表元信息：主键列 + 非主键更新列 + int64 时间字段。
type tableMeta struct {
	pkCols     []clause.Column
	updateCols []string
	createdAt  *schema.Field
	updatedAt  *schema.Field
}

// parseTableMeta 解析 T 的主键列与非主键列。
func parseTableMeta[T PlayerTableModel]() (*tableMeta, error) {
	sch, err := schema.Parse(new(T), &schemaCache, schema.NamingStrategy{})
	if err != nil {
		return nil, errors.New("cache: parse table schema failed: " + err.Error())
	}
	m := &tableMeta{}
	for _, f := range sch.PrimaryFields {
		m.pkCols = append(m.pkCols, clause.Column{Name: f.DBName})
	}
	if len(m.pkCols) == 0 {
		return nil, errors.New("cache: table " + sch.Table + " has no primary key")
	}
	for _, f := range sch.Fields {
		// gorm 只自动填充 time.Time，int64 时间戳需要落库前手动赋值
		switch f.DBName {
		case "created_at":
			if f.FieldType.Kind() == reflect.Int64 {
				m.createdAt = f
			}
		case "updated_at":
			if f.FieldType.Kind() == reflect.Int64 {
				m.updatedAt = f
			}
		}
		// created_at 只在插入时写入，upsert 冲突时保留首次创建时间
		if f.PrimaryKey || f.DBName == "" || f.DBName == "created_at" {
			continue
		}
		m.updateCols = append(m.updateCols, f.DBName)
	}
	return m, nil
}

// fillTimestamps 为 int64 时间字段填毫秒时间戳：
// created_at 仅在为 0（首次插入）时赋值，updated_at 每次落库刷新。
func fillTimestamps[T any](meta *tableMeta, rows []T) {
	if meta.createdAt == nil && meta.updatedAt == nil {
		return
	}
	now := time.Now().UnixMilli()
	ctx := context.Background()
	for i := range rows {
		rv := reflect.ValueOf(&rows[i]).Elem()
		if f := meta.createdAt; f != nil {
			if v, _ := f.ValueOf(ctx, rv); v != nil {
				if n, ok := v.(int64); ok && n == 0 {
					_ = f.Set(ctx, rv, now)
				}
			}
		}
		if f := meta.updatedAt; f != nil {
			_ = f.Set(ctx, rv, now)
		}
	}
}

// upsertRows 批量 upsert：冲突列 = 主键列，更新列 = 非 PK 列（按 chunk 切分）。
func upsertRows[T PlayerTableModel](db *gorm.DB, rows []T, meta *tableMeta) error {
	fillTimestamps(meta, rows)
	for start := 0; start < len(rows); start += upsertChunkSize {
		end := start + upsertChunkSize
		if end > len(rows) {
			end = len(rows)
		}
		if err := db.Clauses(clause.OnConflict{
			Columns:   meta.pkCols,
			DoUpdates: clause.AssignmentColumns(meta.updateCols),
		}).Create(rows[start:end]).Error; err != nil {
			return err
		}
	}
	return nil
}

// NewPlayerTableCache 单行表缓存：key=playerID，value=T（一玩家一行，如 PlayerCore / PlayerProfile）。
// 读：SELECT * WHERE player_id=?，无行返回 gorm.ErrRecordNotFound。
// 写：异步批量 upsert（冲突列取主键，更新列取非 PK 列）。
// 删：DELETE WHERE player_id IN（同步）。
func NewPlayerTableCache[T PlayerTableModel](cfg CacherConfig) *Cache[int64, T] {
	meta, err := parseTableMeta[T]()
	if err != nil {
		panic(err)
	}
	loader := func(ctx context.Context, playerID int64) (T, error) {
		db, err := svc.Ctx().DB()
		if err != nil {
			var zero T
			return zero, err
		}
		var row T
		if err := db.WithContext(ctx).First(&row, "player_id = ?", playerID).Error; err != nil {
			return row, err
		}
		return row, nil
	}
	saver := func(ctx context.Context, entries []CacheEntry[int64, T]) error {
		db, err := svc.Ctx().DB()
		if err != nil {
			return err
		}
		rows := make([]T, 0, len(entries))
		for _, e := range entries {
			rows = append(rows, e.Value)
		}
		return upsertRows(db.WithContext(ctx), rows, meta)
	}
	deleter := func(ctx context.Context, ids []int64) error {
		db, err := svc.Ctx().DB()
		if err != nil {
			return err
		}
		return db.WithContext(ctx).Delete(new(T), "player_id IN ?", ids).Error
	}
	return NewCache[int64, T](cfg, loader, saver, deleter)
}

// NewPlayerRowsCache 多行表缓存：key=playerID，value=[]T（一玩家多行，如 PlayerAsset）。
// 读：SELECT * WHERE player_id=?，无行返回空切片。
// 写/删：同单行表缓存。
func NewPlayerRowsCache[T PlayerTableModel](cfg CacherConfig) *Cache[int64, []T] {
	meta, err := parseTableMeta[T]()
	if err != nil {
		panic(err)
	}
	loader := func(ctx context.Context, playerID int64) ([]T, error) {
		db, err := svc.Ctx().DB()
		if err != nil {
			return nil, err
		}
		var list []T
		if err := db.WithContext(ctx).Find(&list, "player_id = ?", playerID).Error; err != nil {
			return nil, err
		}
		return list, nil
	}
	saver := func(ctx context.Context, entries []CacheEntry[int64, []T]) error {
		db, err := svc.Ctx().DB()
		if err != nil {
			return err
		}
		rows := make([]T, 0, len(entries))
		for _, e := range entries {
			rows = append(rows, e.Value...)
		}
		if len(rows) == 0 {
			return nil
		}
		return upsertRows(db.WithContext(ctx), rows, meta)
	}
	deleter := func(ctx context.Context, ids []int64) error {
		db, err := svc.Ctx().DB()
		if err != nil {
			return err
		}
		return db.WithContext(ctx).Delete(new(T), "player_id IN ?", ids).Error
	}
	return NewCache[int64, []T](cfg, loader, saver, deleter)
}
