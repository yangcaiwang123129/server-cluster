// Package repo 数据访问层：接口 + gorm 实现 + 单例。
package repo

import (
	"context"
	"server-cluster/share/singleton"

	"gorm.io/gorm"

	"server-cluster/internal/svc"
	"server-cluster/share/model"
)

// PlayerRepo 账号数据访问接口。
type PlayerRepo interface {
	// Save 创建账号
	Save(ctx context.Context, m *model.PlayerBase) error
	// FindByUsername 按用户名查询
	FindByUsername(ctx context.Context, username string) (*model.PlayerBase, error)
}

var playerRepo = singleton.NewContainer[PlayerRepo]("PlayerRepo")

func init() {
	playerRepo.Register(NewAccountRepo())
}

// Repo 获取账号数据访问单例。
func Repo() PlayerRepo { return playerRepo.MustGet() }

type playerRepoImpl struct{}

// NewAccountRepo 构造 gorm 实现。
func NewAccountRepo() PlayerRepo { return &playerRepoImpl{} }

func (this *playerRepoImpl) Save(ctx context.Context, m *model.PlayerBase) error {
	db, err := svc.Ctx().DB()
	if err != nil {
		return gorm.ErrInvalidDB
	}
	return db.WithContext(ctx).Save(m).Error
}

func (this *playerRepoImpl) FindByUsername(ctx context.Context, username string) (*model.PlayerBase, error) {
	db, err := svc.Ctx().DB()
	if err != nil {
		return nil, gorm.ErrInvalidDB
	}
	var m *model.PlayerBase
	if err := db.WithContext(ctx).Where("username = ?", username).First(&m).Error; err != nil {
		return nil, err
	}
	return m, nil
}

func (this *playerRepoImpl) FindByPhoneNumber(ctx context.Context, username string) (*model.PlayerBase, error) {
	db, err := svc.Ctx().DB()
	if err != nil {
		return nil, gorm.ErrInvalidDB
	}
	var m *model.PlayerBase
	if err := db.WithContext(ctx).Where("phone_number = ?", username).First(&m).Error; err != nil {
		return nil, err
	}
	return m, nil
}
