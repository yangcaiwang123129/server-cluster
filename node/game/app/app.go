// Package app 装配层：业务接口定义 + 单例注册 + 依赖装配。
package app

import (
	"context"
	"fmt"
	"server-cluster/node/game/app/logic"
	"server-cluster/node/game/app/repo"
	"server-cluster/share/singleton"
	"sync/atomic"

	"github.com/dobyte/due/v2/cluster/node"
	"github.com/dobyte/due/v2/etc"
	"github.com/dobyte/due/v2/log"
	"github.com/dobyte/jwt"

	"server-cluster/internal/db"
	"server-cluster/internal/store"
	"server-cluster/internal/svc"
	"server-cluster/share/cfg"
	"server-cluster/share/model"
	"server-cluster/share/pb"
)

var jwtInst *jwt.JWT

// ParseToken 校验 account 签发的 JWT 并提取 uid。
func ParseToken(token string) (int64, error) {
	if jwtInst == nil {
		return 0, fmt.Errorf("jwt not initialized")
	}
	payload, err := jwtInst.ExtractPayload(token)
	if err != nil {
		return 0, err
	}
	v, ok := payload["uid"]
	if !ok {
		return 0, fmt.Errorf("missing uid in token")
	}
	f, ok := v.(float64)
	if !ok {
		return 0, fmt.Errorf("invalid uid type in token")
	}
	return int64(f), nil
}

// PlayerLogic 玩家模块业务接口。
type PlayerLogic interface {
	// EnterGame 进入游戏
	EnterGame(ctx context.Context, uid int64, req *pb.EnterGameReq) (*pb.EnterGameResp, error)
	// GetPlayerInfo 拉取玩家信息
	GetPlayerInfo(ctx context.Context, uid int64) (*pb.GetPlayerInfoResp, error)
}

var playerLogic = singleton.NewContainer[PlayerLogic]("PlayerLogic")

// RegisterPlayerLogic 装配期注册实现；仅 wire 调用。
func RegisterPlayerLogic(l PlayerLogic) { playerLogic.Register(l) }

// Player 获取玩家模块业务单例。
func Player() PlayerLogic { return playerLogic.MustGet() }

// 玩家数据存储层单例（配置驱动：etc.store 段缺席则不启用）。
var (
	playerStore = singleton.NewContainer[store.PlayerStore]("PlayerStore")
	storeOn     atomic.Bool
)

// StoreEnabled 报告存储层是否启用；业务调用 Store 前须先判断。
func StoreEnabled() bool { return storeOn.Load() }

// Store 获取玩家存储单例；未启用时 panic（先判 StoreEnabled）。
func Store() store.PlayerStore { return playerStore.MustGet() }

// Init 初始化应用：svc（DB/Redis）→ 配置表 → 自动建表 → store（存储层）→ jwt → wire（注册实现）。
// 须在注册路由前调用。
func Init(proxy *node.Proxy) error {
	if err := svc.Init(); err != nil {
		return err
	}
	// 加载 Luban 配置表（JSON 内嵌，失败终止启动）
	if err := cfg.Load(); err != nil {
		return err
	}
	log.Infof("cfg: tables loaded, items=%d", len(cfg.T().TbItem.GetDataList()))
	// 自动构建表（玩家相关表 + 背包）；默认实例未配置（dsn 留空）时跳过
	if inst, err := svc.Ctx().MySQL.Default(); err == nil {
		if err := db.AutoMigrate(inst.Gorm(),
			&model.PlayerModel{},
			&model.PlayerCore{},
			&model.PlayerAsset{},
			&model.PlayerProfile{},
			&model.BackpackRow{},
		); err != nil {
			return err
		}
	}
	if err := initStore(); err != nil {
		return err
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

// Destroy 进程退出后释放存储层、DB / Redis 等资源。
func Destroy() {
	closeStore()
	_ = svc.Close()
}

// initStore 初始化存储层：加载配置 → 组装 → 阻塞式 WAL 恢复 → 注册单例。
// etc.store 段缺席时跳过（模板可无存储启动）。
func initStore() error {
	cfg, err := store.Load()
	if err != nil {
		return err
	}
	if cfg == nil {
		log.Infof("store: etc.store not configured, player store disabled")
		return nil
	}
	ps, err := store.New(context.Background(), cfg)
	if err != nil {
		return err
	}
	// 阻塞式崩溃恢复：WAL 回放落 MySQL 并截断，完成前不对外服务。
	if err := ps.Recover(context.Background()); err != nil {
		_ = ps.Close(context.Background())
		return err
	}
	playerStore.Register(ps)
	storeOn.Store(true)
	log.Infof("store: player store ready, wal_id=%s", cfg.WALID)
	return nil
}

// closeStore 释放存储层资源（最终 flush + 关 WAL / 缓存 / MySQL 连接）。
func closeStore() {
	if !storeOn.CompareAndSwap(true, false) {
		return
	}
	_ = playerStore.MustGet().Close(context.Background())
}

// wire 依赖装配：构造各模块实现并注册到单例容器。
// 注意依赖方向：app -> logic/repo，反向只允许依赖接口，避免循环引用。
func wire() {
	repo.RegisterPlayer(repo.NewPlayerRepo())
	RegisterPlayerLogic(logic.NewPlayerLogic())
}
