package enum

type ItemID int32

const (
	Coin   ItemID = 1 // 巡界币（model.Player.Coin，列 diamond）
	Gold   ItemID = 2 // 巡界金（model.Player.Gold，列 gold）
	Coupon ItemID = 3 // 热源卷（model.Player.Coupon，列 coupon）
)

// CurrencyIDs 全部货币物品配置ID（固定顺序，供遍历查询/列表）。
var CurrencyIDs = []ItemID{Coin, Gold, Coupon}

// CurrencyFields 货币物品配置ID → model.Player 的 Go 字段名；未知配置ID不在表中。
var CurrencyFields = map[ItemID]string{Coin: "Coin", Gold: "Gold", Coupon: "Coupon"}
