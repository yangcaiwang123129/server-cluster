// Package dto HTTP API 出入参（Data Transfer Object）：只放结构体，不放逻辑。
// 客户端 wire 消息不在此定义，统一由 share/proto（生成 share/pb）承载；表模型在 share/model。
package dto

// RegisterReq 注册请求。
type RegisterReq struct {
	Username string `json:"username" validate:"required,min=3,max=32"`
	Password string `json:"password" validate:"required,min=6,max=64"`
}

// RegisterResp 注册响应。
type RegisterResp struct {
	UID      int64  `json:"uid"`
	Username string `json:"username"`
}

// LoginReq 登录请求。
type LoginReq struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// ServerItem 区服信息。
type ServerItem struct {
	ServerID int32  `json:"server_id"`
	Name     string `json:"name"`
	GateAddr string `json:"gate_addr"` // WebSocket 接入地址，如 ws://127.0.0.1:3553
	Status   int32  `json:"status"`    // 0 维护 1 流畅 2 火爆 3 新服
}

// LoginResp 登录响应。
type LoginResp struct {
	UID       int64        `json:"uid"`
	Token     string       `json:"token"`
	ExpiredAt int64        `json:"expired_at"`
	Servers   []ServerItem `json:"servers"`
}

// ServerListResp 区服列表响应。
type ServerListResp struct {
	Servers []ServerItem `json:"servers"`
}
