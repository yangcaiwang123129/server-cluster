package dto

// PlayerRegisterReq 注册请求。
type PlayerRegisterReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// PlayerRegisterResp 注册响应。
type PlayerRegisterResp struct {
	Token    string `json:"token"`
	GateAddr string `json:"gateAddr"`
}

// PlayerLoginReq 登录请求。
type PlayerLoginReq struct {
	LoginType int32 `json:"loginType"` // 登录类型 0账号登录 1手机号登录 2微信授权登录

	Username string `json:"username"`
	Password string `json:"password"`

	Zone        string `json:"zone"`
	PhoneNumber string `json:"phoneNumber"`
}

// PlayerLoginResp 登录响应。
type PlayerLoginResp struct {
	Token    string `json:"token"`
	GateAddr string `json:"gateAddr"`
}
