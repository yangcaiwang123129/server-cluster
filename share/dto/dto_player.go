package dto

// PlayerListReq 玩家列表请求。
type PlayerListReq struct {
	PageReq
	PlayerID       int64
	Username       string
	PhoneNumber    string // 手机号（字符串，支持前缀/模糊）
	Status         int32  // 0 全部 1 正常 2 封号
	CreatedAtStart int64  // 创建时间起（毫秒）
	CreatedAtEnd   int64  // 创建时间止（毫秒）
	Order          string
}

// PlayerResp 玩家账号信息（player 账号与货币 + player_extra 昵称/头像/等级）。
type PlayerResp struct {
	PlayerID    int64  `json:"playerId"`
	Username    string `json:"username"`
	Nickname    string `json:"nickname"`
	AvatarID    string `json:"avatarId"`
	Level       int32  `json:"level"`
	VipLevel    int32  `json:"vipLevel"`
	Zone        int32  `json:"zone"`
	PhoneNumber string `json:"phoneNumber"` // 带区号，如 "+8613800138000"（来自 player_auth）
	Coupon      int64  `json:"coupon"` // 点券
	Gold        int64  `json:"gold"`   // 金
	Coin        int64  `json:"coin"`   // 币
	RegisterIP  string `json:"registerIp"`
	CreatedAt   int64  `json:"createdAt"`
	// 状态 & 封号
	Status    int32  `json:"status"`    // 1 正常 2 封号
	BanUntil  int64  `json:"banUntil"`  // 封号到期时间戳（秒），0 表示永久
	BanReason string `json:"banReason"` // 封号原因
}

// OnlinePlayerResp 在线玩家信息。
type OnlinePlayerResp struct {
	PlayerID     int64  `json:"playerId"`
	EnterTime    int64  `json:"enterTime"`
	LobbyNodeID  string `json:"lobbyNodeId"` // 所在大厅(game)服实例 ID
	LoginIP      string `json:"loginIp"`     // 登录 IP（内存维护，不落库）
}

// PlayerKickReq 踢玩家下线请求。
type PlayerKickReq struct {
	PlayerID int64  `json:"playerId"` // 玩家ID
	NodeID   string `json:"nodeId"`   // 玩家所在 game 服实例 ID（定向调用）
	KickType int32  `json:"kickType"` // 踢下线类型：1 主动踢 2 封号踢
}

// PlayerBanReq 封号请求。
type PlayerBanReq struct {
	PlayerID  int64  `json:"playerId"`           // 玩家ID
	Duration  int32  `json:"duration"`           // 封号时长（秒），0 表示永久
	Reason    string `json:"reason"`             // 封号原因
}

// PlayerUnbanReq 解封请求。
type PlayerUnbanReq struct {
	PlayerID int64 `json:"playerId"` // 玩家ID
}

// PlayerResetPasswordReq 重置密码请求。
type PlayerResetPasswordReq struct {
	PlayerID    int64  `json:"playerId"`    // 玩家ID
	NewPassword string `json:"newPassword"` // 新密码（明文，服务端哈希后存储）
}
