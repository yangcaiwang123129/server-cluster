// Package logic 业务实现：实现 app 定义的接口。
package logic

import (
	"context"
	"errors"
	"server-cluster/share/util"

	"server-cluster/mesh/login/app/repo"
	"server-cluster/share/dto"
	"server-cluster/share/errs"
	"server-cluster/share/model"

	"github.com/dobyte/due/v2/utils/xtime"
	"github.com/dobyte/jwt"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const GateAddr = "127.0.0.1:9000"

type AccountLogic struct {
	jwtInst *jwt.JWT // token 签发器
}

// NewAccountLogic 构造账号业务实现。
func NewAccountLogic(jwtInst *jwt.JWT) *AccountLogic {
	return &AccountLogic{jwtInst: jwtInst}
}

func (l *AccountLogic) Register(ctx context.Context, req *dto.RegisterReq) (*dto.RegisterResp, error) {
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
		ID:        util.SnowflakeNode.Next(),
		PlayerID:  util.SnowflakeNode.Next(),
		Username:  req.Username,
		Password:  string(hash),
		CreatedAt: xtime.Now().UnixMilli(),
	}
	if err := repo.Account().Create(ctx, m); err != nil {
		return nil, err
	}
	// 注册成功后直接自动登录签发 token
	token, err := l.jwtInst.GenerateToken(jwt.Payload{"player_id": m.ID})
	if err != nil {
		return nil, err
	}
	return &dto.RegisterResp{Token: token.Token, GateAddr: GateAddr}, nil
}

func (l *AccountLogic) Login(ctx context.Context, req *dto.LoginReq) (*dto.LoginResp, error) {
	switch req.LoginType {
	case 0: // 账号密码登录
		return l.loginByUsername(ctx, req)
	case 1: // 手机号登录
		return nil, errors.New("该登录方式暂不支持")
	case 2: // 微信授权登录
		return nil, errors.New("该登录方式暂不支持")
	default:
		return nil, errs.ErrInvalidParams
	}
}

// loginByUsername 账号密码登录分支。
func (l *AccountLogic) loginByUsername(ctx context.Context, req *dto.LoginReq) (*dto.LoginResp, error) {
	m, err := repo.Account().FindByUsername(ctx, req.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrAccountNotFound
		}
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(m.Password), []byte(req.Password)); err != nil {
		return nil, errs.ErrPasswordWrong
	}
	token, err := l.jwtInst.GenerateToken(jwt.Payload{"playerID": m.ID})
	if err != nil {
		return nil, err
	}
	return &dto.LoginResp{Token: token.Token, GateAddr: GateAddr}, nil
}
