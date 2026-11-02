package consts

// LoginType 登录类型，对齐 proto enum LoginType。
const (
	// LoginTypeGuest 游客登录（纯 device_id，即登即建号，不入 player_auth 表）。
	LoginTypeGuest int32 = 0
	// LoginTypeAccount 账号密码登录（username + password）。
	LoginTypeAccount int32 = 1
	// LoginTypePhoneSMS 手机号 + 短信验证码。
	LoginTypePhoneSMS int32 = 2
	// LoginTypeWechat 微信授权（code 换 openid）。
	LoginTypeWechat int32 = 3
	// LoginTypeApple Apple ID 登录（identityToken 验签解 sub）。
	LoginTypeApple int32 = 4
)
