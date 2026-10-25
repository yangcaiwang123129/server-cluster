// Package logic 业务实现：实现 app 定义的接口。
package logic

import (
	"context"
	"errors"

	"github.com/dobyte/jwt"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"server-cluster/mesh/login/app/repo"
	"server-cluster/share/dto"
	"server-cluster/share/errs"
	"server-cluster/share/model"
)

type accountLogic struct {
	servers []dto.ServerItem // 区服列表（装配期加载，运行期只读）
	jwtInst *jwt.JWT         // token 签发器
}

// NewAccountLogic 构造账号业务实现；servers 为启动期加载的区服列表。
func NewAccountLogic(servers []dto.ServerItem, jwtInst *jwt.JWT) *accountLogic {
	return &accountLogic{servers: servers, jwtInst: jwtInst}
}

func (l *accountLogic) Register(ctx context.Context, req *dto.RegisterReq) (*dto.RegisterResp, error) {
	// 检查用户名是否已存在
	_, err := repo.Account().FindByUsername(ctx, req.Username)
	if err == nil {
		return nil, errs.ErrAccountExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	// 加密密码
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	m := &model.AccountModel{
		Username:     req.Username,
		PasswordHash: string(hash),
	}
	if err := repo.Account().Create(ctx, m); err != nil {
		return nil, err
	}
	return &dto.RegisterResp{UID: m.ID, Username: m.Username}, nil
}

func (l *accountLogic) Login(ctx context.Context, req *dto.LoginReq) (*dto.LoginResp, error) {
	m, err := repo.Account().FindByUsername(ctx, req.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrAccountNotFound
		}
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(m.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errs.ErrPasswordWrong
	}
	token, err := l.jwtInst.GenerateToken(jwt.Payload{"uid": m.ID})
	if err != nil {
		return nil, err
	}
	return &dto.LoginResp{
		UID:       m.ID,
		Token:     token.Token,
		ExpiredAt: token.ExpiredAt.Unix(),
		Servers:   l.servers,
	}, nil
}

func (l *accountLogic) ServerList(ctx context.Context) (*dto.ServerListResp, error) {
	return &dto.ServerListResp{Servers: l.servers}, nil
}
