package dto

// PlayerListReq 玩家列表请求。
type PlayerListReq struct {
	PageReq
	Username string
	Order    string
}

// PlayerResp 玩家账号信息（player_base）。
type PlayerResp struct {
	PlayerID    int64  `json:"playerId"`
	Username    string `json:"username"`
	Zone        int32  `json:"zone"`
	PhoneNumber int32  `json:"phoneNumber"`
	CreatedAt   int64  `json:"createdAt"`
}

// OnlinePlayerResp 在线玩家信息。
type OnlinePlayerResp struct {
	PlayerID  int64 `json:"playerId"`
	EnterTime int64 `json:"enterTime"`
}
