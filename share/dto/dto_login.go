// Package dto HTTP API 出入参（Data Transfer Object）：只放结构体，不放逻辑。
// 客户端 wire 消息不在此定义，统一由 share/proto（生成 share/pb）承载；表模型在 share/model。
package dto

// RegisterReq 注册请求。
type RegisterReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// RegisterResp 注册响应。
type RegisterResp struct {
	Token    string `json:"token"`
	GateAddr string `json:"gateAddr"`
}

// LoginReq 登录请求。
type LoginReq struct {
	LoginType int32 `json:"loginType"` // 登录类型 0账号登录 1手机号登录 2微信授权登录

	Username string `json:"username"`
	Password string `json:"password"`

	PhoneNumber string `json:"phoneNumber"`
}

// LoginResp 登录响应。
type LoginResp struct {
	Token    string `json:"token"`
	GateAddr string `json:"gateAddr"`
}
