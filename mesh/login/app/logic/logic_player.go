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

	"github.com/dobyte/due/v2/etc"
	"github.com/dobyte/due/v2/utils/xtime"
	"github.com/dobyte/jwt"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// defaultGateAddr 未配置 etc.gate.addr 时的回退网关地址。
const defaultGateAddr = "127.0.0.1:9000"

// gateAddr 下发给客户端的网关地址，优先取 etc.gate.addr。
func gateAddr() string {
	if addr := etc.Get("etc.gate.addr").String(); addr != "" {
		return addr
	}
	return defaultGateAddr
}

type playerBaseLogic struct{}

var playerBaseLogicInst = singleton.NewContainer[global.PlayerLogic]("PlayerBaseLogic")

// init 装配期自注册实现。
func init() {
	playerBaseLogicInst.Register(&playerBaseLogic{})
}

// PlayerBaseLogic 获取账号模块业务单例。
func PlayerBaseLogic() global.PlayerLogic { return playerBaseLogicInst.MustGet() }

func (this *playerBaseLogic) PlayerRegister(ctx context.Context, req *dto.PlayerRegisterReq) (*dto.PlayerRegisterResp, error) {
	// 检查用户名是否已存在
	_, err := repo.Repo().FindByUsername(ctx, req.Username)
	if err == nil {
		return nil, errs.ErrPlayerExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	// 加密密码
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	m := &model.PlayerBase{
		PlayerID:  svc.NextID(),
		Username:  req.Username,
		Password:  string(hash),
		CreatedAt: xtime.Now().UnixMilli(),
	}
	if err := repo.Repo().Save(ctx, m); err != nil {
		return nil, err
	}
	// 注册成功后直接自动登录签发 token
	token, err := svc.JWT().GenerateToken(jwt.Payload{svc.PayloadPlayerID: m.PlayerID})
	if err != nil {
		return nil, err
	}
	return &dto.PlayerRegisterResp{Token: token.Token, GateAddr: gateAddr()}, nil
}

func (this *playerBaseLogic) PlayerLogin(ctx context.Context, req *dto.PlayerLoginReq) (*dto.PlayerLoginResp, error) {
	switch req.LoginType {
	case dto.LoginTypePassword:
		return this.loginByUsername(ctx, req)
	case dto.LoginTypePhone:
		return nil, errors.New("该登录方式暂不支持")
	case dto.LoginTypeWechat:
		return nil, errors.New("该登录方式暂不支持")
	default:
		return nil, errs.ErrInvalidParams
	}
}

// loginByUsername 账号密码登录分支。
func (this *playerBaseLogic) loginByUsername(ctx context.Context, req *dto.PlayerLoginReq) (*dto.PlayerLoginResp, error) {
	m, err := repo.Repo().FindByUsername(ctx, req.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrPlayerNotFound
		}
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(m.Password), []byte(req.Password)); err != nil {
		return nil, errs.ErrPasswordWrong
	}
	token, err := svc.JWT().GenerateToken(jwt.Payload{svc.PayloadPlayerID: m.PlayerID})
	if err != nil {
		return nil, err
	}
	return &dto.PlayerLoginResp{Token: token.Token, GateAddr: gateAddr()}, nil
}
