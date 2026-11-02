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
	Save(ctx context.Context, m *model.Player) error
	// FindByUsername 按用户名查询
	FindByUsername(ctx context.Context, username string) (*model.Player, error)
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

func (this *playerRepoImpl) Save(ctx context.Context, m *model.Player) error {
	db, err := svc.Ctx().DB()
	if err != nil {
		return gorm.ErrInvalidDB
	}
	// 注册永远是新玩家，用 Create 保证 BeforeCreate hook 正确设置 CreatedAt/UpdatedAt。
	// GORM 的 Save 对带主键的 struct 会先查后存，新行也可能跳过 BeforeCreate。
	return db.WithContext(ctx).Create(m).Error
}

func (this *playerRepoImpl) FindByUsername(ctx context.Context, username string) (*model.Player, error) {
	db, err := svc.Ctx().DB()
	if err != nil {
		return nil, gorm.ErrInvalidDB
	}
	var m *model.Player
	if err := db.WithContext(ctx).Where("username = ?", username).First(&m).Error; err != nil {
		return nil, err
	}
	return m, nil
}

func (this *playerRepoImpl) FindByPhoneNumber(ctx context.Context, username string) (*model.Player, error) {
	db, err := svc.Ctx().DB()
	if err != nil {
		return nil, gorm.ErrInvalidDB
	}
	var m *model.Player
	if err := db.WithContext(ctx).Where("phone_number = ?", username).First(&m).Error; err != nil {
		return nil, err
	}
	return m, nil
}
