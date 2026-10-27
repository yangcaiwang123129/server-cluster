// Package repo 数据访问层：接口 + gorm 实现 + 单例。
package repo

import (
	"context"
	"server-cluster/share/singleton"

	"gorm.io/gorm"

	"server-cluster/internal/svc"
	"server-cluster/share/model"
)

// AccountRepo 账号数据访问接口。
type AccountRepo interface {
	// Create 创建账号
	Create(ctx context.Context, m *model.AccountModel) error
	// FindByUsername 按用户名查询
	FindByUsername(ctx context.Context, username string) (*model.AccountModel, error)
}

var accountRepo = singleton.NewContainer[AccountRepo]("AccountRepo")

// init 装配期自注册实现。
func init() {
	accountRepo.Register(NewAccountRepo())
}

// Account 获取账号数据访问单例。
func Account() AccountRepo { return accountRepo.MustGet() }

type accountRepoImpl struct{}

// NewAccountRepo 构造 gorm 实现。
func NewAccountRepo() AccountRepo { return &accountRepoImpl{} }

func (r *accountRepoImpl) Create(ctx context.Context, m *model.AccountModel) error {
	db, err := svc.Ctx().DB()
	if err != nil {
		return gorm.ErrInvalidDB
	}
	return db.WithContext(ctx).Create(m).Error
}

func (r *accountRepoImpl) FindByUsername(ctx context.Context, username string) (*model.AccountModel, error) {
	db, err := svc.Ctx().DB()
	if err != nil {
		return nil, gorm.ErrInvalidDB
	}
	var m *model.AccountModel
	if err := db.WithContext(ctx).Where("username = ?", username).First(&m).Error; err != nil {
		return nil, err
	}
	return m, nil
}
