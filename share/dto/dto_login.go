package dto

// PlayerLoginReq 统一登录请求，按 loginType 填充不同字段。
type PlayerLoginReq struct {
	LoginType int32 `json:"loginType"` // 必填，0~4

	// GUEST(0)
	// 所有类型通用：客户端生成的 UUIDv4 设备 ID
	DeviceId string `json:"deviceId"` // 必填

	// ACCOUNT(1)
	Username string `json:"username"`
	Password string `json:"password"`

	// PHONE_SMS(2)
	Zone        int32  `json:"zone"`
	PhoneNumber string `json:"phoneNumber"` // 手机号
	SmsCode     string `json:"smsCode"`     // 6 位验证码

	// WECHAT(3)
	WechatCode string `json:"wechatCode"` // wx.login() 返回的 code

	// APPLE(4)
	AppleIdentityToken string `json:"appleIdentityToken"` // Sign in with Apple 的 identityToken (JWT)
	AppleNonce         string `json:"appleNonce"`         // 登录时传给 Apple 的原始 nonce，防止重放
}

// PlayerLoginResp 统一登录响应。
type PlayerLoginResp struct {
	Token       string `json:"token"`       // JWT，payload 只有 player_id 字符串
	GateAddr    string `json:"gateAddr"`    // 网关连接地址，如 "drpc://xxx:yyy"
	IsNewPlayer bool   `json:"isNewPlayer"` // 是否本次新建（客户端决定是否弹新手引导）
}

// PlayerBindReq 绑定新凭据（游客升级 / 已登录玩家加绑）。
type PlayerBindReq struct {
	PlayerId           int64  `json:"playerId"`
	AuthType           int32  `json:"authType"` // 1账号 2手机 3微信 4Apple
	Username           string `json:"username"`
	Password           string `json:"password"`
	PhoneNumber        string `json:"phoneNumber"`
	SmsCode            string `json:"smsCode"`
	WechatCode         string `json:"wechatCode"`
	AppleIdentityToken string `json:"appleIdentityToken"`
	AppleNonce         string `json:"appleNonce"`
}

// PlayerPhoneSendCodeReq 发送短信验证码。
type PlayerPhoneSendCodeReq struct {
	PhoneNumber string `json:"phoneNumber"` // 带区号，如 "+8613800138000"
}
