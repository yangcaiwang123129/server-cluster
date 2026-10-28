package dto

// ItemListReq 物品配置列表请求（读 Luban 配置，内存分页）。
type ItemListReq struct {
	PageReq
	Keyword string
}

// ItemCfgResp 物品配置行。
type ItemCfgResp struct {
	ID              int32  `json:"id"`
	Name            string `json:"name"`
	Desc            string `json:"desc"`
	Type            int32  `json:"type"`
	IconKey         string `json:"iconKey"`
	Width           int32  `json:"width"`
	Height          int32  `json:"height"`
	Rarity          int32  `json:"rarity"`
	MaxCount        int32  `json:"maxCount"`
	OverlayMaxCount int32  `json:"overlayMaxCount"`
	StorageType     int32  `json:"storageType"`
	Price           int32  `json:"price"`
	SellingPrice    int32  `json:"sellingPrice"`
	UsedInBackpack  bool   `json:"usedInBackpack"`
}

// PlayerContainerReq 玩家容器查询请求。
type PlayerContainerReq struct {
	PlayerID      int64 `json:"playerId"`
	ContainerType int32 `json:"containerType"` // 1 背包 2 仓库 3 保险箱
}

// ContainerItemResp 容器内物品（含配置展示信息）。
type ContainerItemResp struct {
	ItemID    int64  `json:"itemId"`
	ItemCfgID int32  `json:"itemCfgId"`
	Count     int32  `json:"count"`
	GridX     int32  `json:"gridX"`
	GridY     int32  `json:"gridY"`
	Rotated   bool   `json:"rotated"`
	Name      string `json:"name"`
	IconKey   string `json:"iconKey"`
	Width     int32  `json:"width"`  // 当前朝向的占格宽
	Height    int32  `json:"height"` // 当前朝向的占格高
	Rarity    int32  `json:"rarity"`
	MaxCount  int32  `json:"maxCount"`
}

// PlayerContainerResp 玩家容器详情。
type PlayerContainerResp struct {
	PlayerID      int64               `json:"playerId"`
	ContainerType int32               `json:"containerType"`
	GridW         int32               `json:"gridW"`
	GridH         int32               `json:"gridH"`
	Version       int32               `json:"version"`
	Items         []ContainerItemResp `json:"items"`
}

// ContainerAddItemReq 向容器放入物品（坐标为左上角格）。
type ContainerAddItemReq struct {
	PlayerID      int64 `json:"playerId"`
	ContainerType int32 `json:"containerType"`
	ItemCfgID     int32 `json:"itemCfgId"`
	Count         int32 `json:"count"`
	GridX         int32 `json:"gridX"`
	GridY         int32 `json:"gridY"`
	Rotated       bool  `json:"rotated"`
}

// ContainerMoveItemReq 移动/旋转容器内物品。
type ContainerMoveItemReq struct {
	PlayerID      int64 `json:"playerId"`
	ContainerType int32 `json:"containerType"`
	ItemID        int64 `json:"itemId"`
	GridX         int32 `json:"gridX"`
	GridY         int32 `json:"gridY"`
	Rotated       bool  `json:"rotated"`
}
