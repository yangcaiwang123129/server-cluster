package enum

// AssetType 玩家资产类型：与 player_asset.asset_type 对应，一行一种资产。
type AssetType int32

const (
	AssetTypeUnspecified AssetType = 0 // 未指定
	AssetTypeGold        AssetType = 1 // 金币
	AssetTypeDiamond     AssetType = 2 // 钻石
	AssetTypePointCoupon AssetType = 3 // 点券
	AssetTypeCoin        AssetType = 4 // 代币
)

// String 返回资产类型名称。
func (t AssetType) String() string {
	switch t {
	case AssetTypeGold:
		return "gold"
	case AssetTypeDiamond:
		return "diamond"
	case AssetTypePointCoupon:
		return "point_coupon"
	case AssetTypeCoin:
		return "coin"
	default:
		return "unspecified"
	}
}
