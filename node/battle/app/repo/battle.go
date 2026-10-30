// Package repo 数据访问层：接口 + gorm 实现 + 单例。
package repo

import (
	"context"
	"server-cluster/share/singleton"
)

// BattleRepo 战斗数据访问接口。
type BattleRepo interface {
	// SaveRecord 保存战斗记录（预留）
	SaveRecord(ctx context.Context, record interface{}) error
}

var battleRepo = singleton.NewContainer[BattleRepo]("BattleRepo")

func init() {
	battleRepo.Register(NewBattleRepo())
}

// Battle 获取战斗数据访问单例。
func Battle() BattleRepo { return battleRepo.MustGet() }

type battleRepoImpl struct{}

// NewBattleRepo 构造 gorm 实现。
func NewBattleRepo() BattleRepo { return &battleRepoImpl{} }

func (r *battleRepoImpl) SaveRecord(ctx context.Context, record interface{}) error {
	return nil
}
