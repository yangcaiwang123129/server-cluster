// Package util gorm 查询辅助：模糊匹配、排序回退、泛型分页查询与分页响应体。
package util

import (
	"gorm.io/gorm"
)

// 分页参数默认值/上限。
const (
	DefaultPageSize = 10
	MaxPageSize     = 10000
)

// PageResp 分页响应体。
type PageResp[T any] struct {
	List      []T   `json:"list"`
	Count     int64 `json:"count"`
	PageIndex int   `json:"pageIndex"`
	PageSize  int   `json:"pageSize"`
}

// PageQuery 分页查询：先 count 再 offset/limit 取数；order 为空时回退 fallback。
// db 须已带 Model 与过滤条件。
func PageQuery[T any](db *gorm.DB, pageIndex, pageSize int, order, fallback string) ([]T, int64, error) {
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset, limit := NormalizePage(pageIndex, pageSize)
	var list []T
	if err := db.Order(DefaultOrder(order, fallback)).Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// NormalizePage 归一化分页参数。
func NormalizePage(pageIndex, pageSize int) (offset, limit int) {
	if pageIndex <= 0 {
		pageIndex = 1
	}
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	return (pageIndex - 1) * pageSize, pageSize
}

// Like 构造模糊匹配值。
func Like(v string) string { return "%" + v + "%" }

// DefaultOrder 排序串为空时回退默认排序。
func DefaultOrder(order, fallback string) string {
	if order == "" {
		return fallback
	}
	return order
}
