// Package app 装配层：业务接口定义 + 单例注册 + 依赖装配。
package app

import (
	"context"
	"server-cluster/share/singleton"

	"github.com/dobyte/due/v2/etc"
	"github.com/dobyte/jwt"

	"server-cluster/internal/db"
	"server-cluster/internal/svc"
	"server-cluster/mesh/login/app/logic"
	"server-cluster/mesh/login/app/repo"
	"server-cluster/share/dto"
	"server-cluster/share/model"
)

var jwtInst *jwt.JWT

// AccountLogic 账号模块业务接口。
type AccountLogic interface {
	// Register 注册账号
	Register(ctx context.Context, req *dto.RegisterReq) (*dto.RegisterResp, error)
	// Login 登录并签发 token
	Login(ctx context.Context, req *dto.LoginReq) (*dto.LoginResp, error)
}

var accountLogic = singleton.NewContainer[AccountLogic]("AccountLogic")

// RegisterAccountLogic 装配期注册实现；仅 wire 调用。
func RegisterAccountLogic(l AccountLogic) { accountLogic.Register(l) }

// Account 获取账号模块业务单例。
func Account() AccountLogic { return accountLogic.MustGet() }

// Init 初始化应用：svc（DB/Redis）→ 自动建表 → jwt → wire（注册实现）。
func Init() error {
	if err := svc.Init(); err != nil {
		return err
	}
	// 自动构建表（账号表）；默认实例未配置（dsn 留空）时跳过
	if inst, err := svc.Ctx().MySQL.Default(); err == nil {
		if err := db.AutoMigrate(inst.Gorm(), &model.AccountModel{}); err != nil {
			return err
		}
	}
	secret := etc.Get("etc.jwt.secret").String()
	validDuration := etc.Get("etc.jwt.valid_duration", 7200).Int()
	inst, err := jwt.NewJWT(
		jwt.WithSecretKey(secret),
		jwt.WithSignAlgorithm(jwt.HS256),
		jwt.WithValidDuration(validDuration),
	)
	if err != nil {
		return err
	}
	jwtInst = inst
	wire()
	return nil
}

// Destroy 进程退出后释放 DB / Redis 等资源。
func Destroy() {
	_ = svc.Close()
}

// wire 依赖装配：构造各模块实现并注册到单例容器。
// 注意依赖方向：app -> logic/repo，反向只允许依赖接口，避免循环引用。
func wire() {
	repo.RegisterAccount(repo.NewAccountRepo())
	RegisterAccountLogic(logic.NewAccountLogic(loadServers(), jwtInst))
}

// loadServers 装配期从 etc.login.servers 加载区服列表（缺省时回退默认区服，保证客户端可连接）。
func loadServers() []dto.ServerItem {
	var servers []dto.ServerItem
	if err := etc.Get("etc.login.servers").Scan(&servers); err != nil || len(servers) == 0 {
		return []dto.ServerItem{{
			ServerID: 1,
			Name:     "默认区",
			GateAddr: "ws://127.0.0.1:3553",
			Status:   1,
		}}
	}
	return servers
}
