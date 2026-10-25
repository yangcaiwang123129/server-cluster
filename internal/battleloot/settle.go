package battleloot

import (
	"server-cluster/share/model"
	"sort"

	"github.com/dobyte/due/v2/log"
)

// AssetMapper 道具 -> 资产类型映射；ok=false 表示该道具不映射为持久资产。
type AssetMapper func(itemID int64) (assetType int32, ok bool)

// MapToAssetChanges 把拾取列表映射为资产变更：itemID -> asset_type，count -> amount，
// 同 assetType 聚合。无法映射的道具记告警并跳过（非持久道具）。
func MapToAssetChanges(items []model.LootItem, mapper AssetMapper) []model.AssetChange {
	if len(items) == 0 {
		return nil
	}
	agg := make(map[int32]int64)
	for _, item := range items {
		assetType, ok := mapper(item.ItemID)
		if !ok {
			log.Warnf("battleloot: item %d has no asset mapping, skipped", item.ItemID)
			continue
		}
		agg[assetType] += item.Count
	}
	changes := make([]model.AssetChange, 0, len(agg))
	for assetType, amount := range agg {
		if amount == 0 {
			continue
		}
		changes = append(changes, model.AssetChange{AssetType: assetType, Amount: amount})
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].AssetType < changes[j].AssetType })
	return changes
}
