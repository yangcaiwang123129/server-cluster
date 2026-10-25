// Package repo 管理后台数据访问层：gorm 查询与少量一致性规则（角色菜单、接口放行规则同步）。
package repo

import (
	"gorm.io/gorm"
)

// Repo 管理后台数据访问单例。
type Repo struct {
	db *gorm.DB
}

var inst *Repo

// Init 绑定数据库句柄；app.Init 在 AutoMigrate 之后调用。
func Init(db *gorm.DB) { inst = &Repo{db: db} }

// R 获取数据访问单例。
func R() *Repo { return inst }

// Available 数据库是否已配置并完成初始化。
func (r *Repo) Available() bool { return r != nil && r.db != nil }

// normalizePage 归一化分页参数。
func normalizePage(pageIndex, pageSize int) (offset, limit int) {
	if pageIndex <= 0 {
		pageIndex = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 10000 {
		pageSize = 10000
	}
	return (pageIndex - 1) * pageSize, pageSize
}

// like 构造模糊匹配值。
func like(v string) string { return "%" + v + "%" }

// defaultOrder 排序串为空时回退默认排序。
func defaultOrder(order, fallback string) string {
	if order == "" {
		return fallback
	}
	return order
}
