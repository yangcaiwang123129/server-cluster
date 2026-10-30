package consts

// 玩家数据默认值。
const (
	// DefaultNicknamePrefix 首次进入游戏时自动昵称的前缀，后接玩家 ID。
	DefaultNicknamePrefix = "player_"
	// DefaultPlayerLevel 新建档玩家的初始等级。
	DefaultPlayerLevel int32 = 1
	// DefaultPlayerSettingsJSON 新建档玩家的个性化设置（MySQL json 列，空串非法）。
	DefaultPlayerSettingsJSON = "{}"
	// ContainerTypeStride 物品容器实例 ID 步长：ID = playerID*Stride + 物品容器类型。
	ContainerTypeStride int64 = 10
)
