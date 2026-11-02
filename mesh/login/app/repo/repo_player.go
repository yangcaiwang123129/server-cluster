// Package repo 数据访问层：接口 + gorm 实现 + 单例。
package repo

import (
	"context"
	"server-cluster/internal/svc"
	"server-cluster/share/model"
	"server-cluster/share/singleton"

	"gorm.io/gorm"
)

// PlayerRepo 玩家基础表数据访问（只管 player 表，不管凭据）。
type PlayerRepo interface {
	Create(ctx context.Context, m *model.Player) error
	GetByID(ctx context.Context, playerID int64) (*model.Player, error)
	GetByDeviceID(ctx context.Context, deviceID string) (*model.Player, error)
	UpdateStatus(ctx context.Context, playerID int64, status int32) error
}

var playerRepo = singleton.NewContainer[PlayerRepo]("PlayerRepo")

func init() { playerRepo.Register(newPlayerRepo()) }

func Repo() PlayerRepo { return playerRepo.MustGet() }

type playerRepoImpl struct{}

func newPlayerRepo() PlayerRepo { return &playerRepoImpl{} }

func (this *playerRepoImpl) Create(ctx context.Context, m *model.Player) error {
	db, err := svc.Ctx().DB()
	if err != nil {
		return gorm.ErrInvalidDB
	}
	return db.WithContext(ctx).Create(m).Error
}

func (this *playerRepoImpl) GetByID(ctx context.Context, playerID int64) (*model.Player, error) {
	db, err := svc.Ctx().DB()
	if err != nil {
		return nil, gorm.ErrInvalidDB
	}
	var m model.Player
	if err := db.WithContext(ctx).Where("player_id = ?", playerID).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (this *playerRepoImpl) GetByDeviceID(ctx context.Context, deviceID string) (*model.Player, error) {
	db, err := svc.Ctx().DB()
	if err != nil {
		return nil, gorm.ErrInvalidDB
	}
	var m model.Player
	if err := db.WithContext(ctx).Where("device_id = ?", deviceID).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (this *playerRepoImpl) UpdateStatus(ctx context.Context, playerID int64, status int32) error {
	db, err := svc.Ctx().DB()
	if err != nil {
		return gorm.ErrInvalidDB
	}
	return db.WithContext(ctx).Model(&model.Player{}).
		Where("player_id = ?", playerID).
		Update("status", status).Error
}

// PlayerAuthRepo 玩家凭据表数据访问（只管 player_auth 表）。
type PlayerAuthRepo interface {
	Create(ctx context.Context, m *model.PlayerAuth) error
	FindByTypeAndIdentifier(ctx context.Context, authType int32, identifier string) (*model.PlayerAuth, error)
	FindByPlayerID(ctx context.Context, playerID int64) ([]*model.PlayerAuth, error)
}

var playerAuthRepo = singleton.NewContainer[PlayerAuthRepo]("PlayerAuthRepo")

func init() { playerAuthRepo.Register(newPlayerAuthRepo()) }

func AuthRepo() PlayerAuthRepo { return playerAuthRepo.MustGet() }

type playerAuthRepoImpl struct{}

func newPlayerAuthRepo() PlayerAuthRepo { return &playerAuthRepoImpl{} }

func (this *playerAuthRepoImpl) Create(ctx context.Context, m *model.PlayerAuth) error {
	db, err := svc.Ctx().DB()
	if err != nil {
		return gorm.ErrInvalidDB
	}
	return db.WithContext(ctx).Create(m).Error
}

func (this *playerAuthRepoImpl) FindByTypeAndIdentifier(ctx context.Context, authType int32, identifier string) (*model.PlayerAuth, error) {
	db, err := svc.Ctx().DB()
	if err != nil {
		return nil, gorm.ErrInvalidDB
	}
	var m model.PlayerAuth
	if err := db.WithContext(ctx).
		Where("auth_type = ? AND auth_identifier = ?", authType, identifier).
		First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (this *playerAuthRepoImpl) FindByPlayerID(ctx context.Context, playerID int64) ([]*model.PlayerAuth, error) {
	db, err := svc.Ctx().DB()
	if err != nil {
		return nil, gorm.ErrInvalidDB
	}
	var list []*model.PlayerAuth
	if err := db.WithContext(ctx).Where("player_id = ?", playerID).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}
