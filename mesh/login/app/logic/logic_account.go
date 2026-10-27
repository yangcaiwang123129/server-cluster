// Package logic 业务实现：实现 global 定义的接口；单例容器在本包，init 自注册（替代 app/wire）。
package logic

import (
	"context"
	"errors"

	"server-cluster/internal/svc"
	"server-cluster/mesh/login/app/global"
	"server-cluster/mesh/login/app/repo"
	"server-cluster/share/dto"
	"server-cluster/share/errs"
	"server-cluster/share/model"
	"server-cluster/share/singleton"

	"github.com/dobyte/due/v2/utils/xtime"
	"github.com/dobyte/jwt"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const GateAddr = "127.0.0.1:9000"

type accountLogic struct{}

var accountLogicInst = singleton.NewContainer[global.AccountLogic]("AccountLogic")

// init 装配期自注册实现。
func init() {
	accountLogicInst.Register(&accountLogic{})
}

// Account 获取账号模块业务单例。
func Account() global.AccountLogic { return accountLogicInst.MustGet() }

func (this *accountLogic) Register(ctx context.Context, req *dto.RegisterReq) (*dto.RegisterResp, error) {
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
		ID:        svc.NextID(),
		PlayerID:  svc.NextID(),
		Username:  req.Username,
		Password:  string(hash),
		CreatedAt: xtime.Now().UnixMilli(),
	}
	if err := repo.Account().Create(ctx, m); err != nil {
		return nil, err
	}
	// 注册成功后直接自动登录签发 token
	token, err := svc.JWT().GenerateToken(jwt.Payload{"player_id": m.ID})
	if err != nil {
		return nil, err
	}
	return &dto.RegisterResp{Token: token.Token, GateAddr: GateAddr}, nil
}

func (this *accountLogic) Login(ctx context.Context, req *dto.LoginReq) (*dto.LoginResp, error) {
	switch req.LoginType {
	case 0: // 账号密码登录
		return this.loginByUsername(ctx, req)
	case 1: // 手机号登录
		return nil, errors.New("该登录方式暂不支持")
	case 2: // 微信授权登录
		return nil, errors.New("该登录方式暂不支持")
	default:
		return nil, errs.ErrInvalidParams
	}
}

// loginByUsername 账号密码登录分支。
func (this *accountLogic) loginByUsername(ctx context.Context, req *dto.LoginReq) (*dto.LoginResp, error) {
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
	token, err := svc.JWT().GenerateToken(jwt.Payload{"playerID": m.ID})
	if err != nil {
		return nil, err
	}
	return &dto.LoginResp{Token: token.Token, GateAddr: GateAddr}, nil
}
