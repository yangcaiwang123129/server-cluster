package dto

// PlayerListReq 玩家列表请求。
type PlayerListReq struct {
	PageReq
	Username string
	Order    string
}

// PlayerResp 玩家账号信息（player 账号与货币 + player_extra 昵称）。
type PlayerResp struct {
	PlayerID    int64  `json:"playerId"`
	Username    string `json:"username"`
	Nickname    string `json:"nickname"`
	Zone        int32  `json:"zone"`
	PhoneNumber int32  `json:"phoneNumber"`
	Coupon      int64  `json:"coupon"` // 点券
	Gold        int64  `json:"gold"`   // 金
	Coin        int64  `json:"coin"`   // 币
	CreatedAt   int64  `json:"createdAt"`
}

// OnlinePlayerResp 在线玩家信息。
type OnlinePlayerResp struct {
	PlayerID  int64  `json:"playerId"`
	EnterTime int64  `json:"enterTime"`
	NodeID    string `json:"nodeId"` // 所在 game 服实例 ID
}

// PlayerKickReq 踢玩家下线请求。
type PlayerKickReq struct {
	PlayerID int64  `json:"playerId"` // 玩家ID
	NodeID   string `json:"nodeId"`   // 玩家所在 game 服实例 ID（定向调用）
	KickType int32  `json:"kickType"` // 踢下线类型：1 主动踢 2 封号踢
}
