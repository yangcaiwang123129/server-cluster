package global

import (
	"server-cluster/internal/cache"
	"server-cluster/share/model"
)

// 玩家数据缓存：读走 LRU（miss 回源 MySQL），写异步批量 upsert。
var (
	// PlayerBaseCache 玩家账号（player_base，一玩家一行）
	PlayerBaseCache = cache.NewPlayerTableCache[model.PlayerBase](cache.CacherConfig{Name: "player_base"})
	// PlayerCoreCache 玩家核心数值（player_core，一玩家一行）
	PlayerCoreCache = cache.NewPlayerTableCache[model.PlayerExtra](cache.CacherConfig{Name: "player_core"})
	// PlayerAssetCache 玩家资产（player_asset，一玩家多行）
	PlayerAssetCache = cache.NewPlayerRowsCache[model.PlayerAsset](cache.CacherConfig{Name: "player_asset"})
	// PlayerHeroCache 玩家英雄持有（player_hero，一玩家一行，英雄 map JSON）
	PlayerHeroCache = cache.NewPlayerTableCache[model.PlayerHero](cache.CacherConfig{Name: "player_hero"})
	// PlayerSkillCache 玩家技能持有（player_skill，一玩家一行，技能 map JSON）
	PlayerSkillCache = cache.NewPlayerTableCache[model.PlayerSkill](cache.CacherConfig{Name: "player_skill"})
	// PlayerWeaponCache 玩家武器持有（player_weapon，一玩家一行，武器 map JSON）
	PlayerWeaponCache = cache.NewPlayerTableCache[model.PlayerWeapon](cache.CacherConfig{Name: "player_weapon"})
	// PlayerEquipCache 玩家装备持有（player_equip，一玩家一行，装备 map JSON）
	PlayerEquipCache = cache.NewPlayerTableCache[model.PlayerEquip](cache.CacherConfig{Name: "player_equip"})
	// PlayerContainerCache 玩家容器（player_container，一玩家一行）
	PlayerContainerCache = cache.NewPlayerTableCache[model.PlayerContainer](cache.CacherConfig{Name: "player_container"})
)

// CloseCaches 停止后台落库协程，把剩余待写数据刷入 MySQL。
func CloseCaches() {
	PlayerBaseCache.Close()
	PlayerCoreCache.Close()
	PlayerAssetCache.Close()
	PlayerHeroCache.Close()
	PlayerSkillCache.Close()
	PlayerWeaponCache.Close()
	PlayerEquipCache.Close()
	PlayerContainerCache.Close()
}
