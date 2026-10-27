package global

import (
	"context"
	"sync/atomic"

	"github.com/dobyte/due/v2/log"

	"server-cluster/internal/store"
	"server-cluster/share/singleton"
)

// 玩家数据存储层单例（配置驱动：etc.store 段缺席则不启用）。
var (
	playerStore = singleton.NewContainer[store.PlayerStore]("PlayerStore")
	storeOn     atomic.Bool
)

// StoreEnabled 报告存储层是否启用；业务调用 Store 前须先判断。
func StoreEnabled() bool { return storeOn.Load() }

// Store 获取玩家存储单例；未启用时 panic（先判 StoreEnabled）。
func Store() store.PlayerStore { return playerStore.MustGet() }

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
