package logic

import (
	"sync"

	"server-cluster/share/tpl"
)

// 武器/技能固定绑定英雄：按英雄配置建立 配置ID → 英雄ID 的反向索引，
// 用于升级请求只携带武器/技能配置ID时反查所属英雄。
var (
	heroIndexOnce   sync.Once
	weaponHeroIndex map[int32]int32
	skillHeroIndex  map[int32]int32
)

// buildHeroIndex 遍历英雄配置建立反向索引；配置重复时以先出现者为准。
func buildHeroIndex() {
	weaponHeroIndex = make(map[int32]int32)
	skillHeroIndex = make(map[int32]int32)
	for heroID, cfg := range tpl.T().RoleInfo.GetDataMap() {
		if cfg == nil {
			continue
		}
		if cfg.WeaponId > 0 {
			if _, ok := weaponHeroIndex[cfg.WeaponId]; !ok {
				weaponHeroIndex[cfg.WeaponId] = heroID
			}
		}
		for _, skillID := range cfg.SkillID {
			if skillID <= 0 {
				continue
			}
			if _, ok := skillHeroIndex[skillID]; !ok {
				skillHeroIndex[skillID] = heroID
			}
		}
	}
}

// heroIDByWeapon 按武器配置ID反查所属英雄ID。
func heroIDByWeapon(cfgID int32) (int32, bool) {
	heroIndexOnce.Do(buildHeroIndex)
	heroID, ok := weaponHeroIndex[cfgID]
	return heroID, ok
}

// heroIDBySkill 按技能配置ID反查所属英雄ID。
func heroIDBySkill(cfgID int32) (int32, bool) {
	heroIndexOnce.Do(buildHeroIndex)
	heroID, ok := skillHeroIndex[cfgID]
	return heroID, ok
}
