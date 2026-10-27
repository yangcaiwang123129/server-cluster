package global

import (
	"server-cluster/internal/cache"
	"server-cluster/share/model"
)

// 玩家数据缓存：读走 LRU（miss 回源 MySQL），写异步批量 upsert。
var (
	// PlayerBaseCache 玩家账号（player_base，一玩家一行）
	PlayerBaseCache = cache.NewPlayerTableCache[model.PlayerBase](cache.CacherConfig{Name: "player_base"})
	// PlayerCoreCache 玩家核心数据（player_core，一玩家一行）
	PlayerCoreCache = cache.NewPlayerTableCache[model.PlayerCore](cache.CacherConfig{Name: "player_core"})
	// PlayerExtraCache 玩家拓展数据（player_extra，一玩家一行）
	PlayerExtraCache = cache.NewPlayerTableCache[model.PlayerExtra](cache.CacherConfig{Name: "player_extra"})
	// PlayerAssetsCache 玩家资产（player_assets，一玩家多行）
	PlayerAssetsCache = cache.NewPlayerRowsCache[model.PlayerAsset](cache.CacherConfig{Name: "player_assets"})
	// PlayerHeroCache 玩家英雄（player_hero，一玩家一行）
	PlayerHeroCache = cache.NewPlayerTableCache[model.PlayerHero](cache.CacherConfig{Name: "player_hero"})
	// PlayerContainerCache 玩家容器（player_container，一玩家一行）
	PlayerContainerCache = cache.NewPlayerTableCache[model.PlayerContainer](cache.CacherConfig{Name: "player_container"})
)

// CloseCaches 停止后台落库协程，把剩余待写数据刷入 MySQL。
func CloseCaches() {
	PlayerBaseCache.Close()
	PlayerCoreCache.Close()
	PlayerExtraCache.Close()
	PlayerAssetsCache.Close()
	PlayerHeroCache.Close()
	PlayerContainerCache.Close()
}
