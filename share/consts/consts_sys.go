package consts

// 管理后台系统常量：角色标识、通用状态、是否标识与系统配置键。
const (
	// AdminRoleKey 超级管理员角色标识：持有全部权限，无需菜单授权。
	AdminRoleKey = "admin"

	// StatusDisabled 停用状态。
	StatusDisabled = "1"
	// StatusEnabled 正常状态。
	StatusEnabled = "2"

	// LogStatusFail 登录/操作日志：失败。
	LogStatusFail = "1"
	// LogStatusSuccess 登录/操作日志：成功。
	LogStatusSuccess = "2"

	// FlagYes 是否标识：是。
	FlagYes = "Y"
	// FlagNo 是否标识：否。
	FlagNo = "N"

	// ConfigKeyAppName 应用名称配置键。
	ConfigKeyAppName = "sys_app_name"
	// ConfigKeyAppLogo 应用Logo配置键。
	ConfigKeyAppLogo = "sys_app_logo"
	// ConfigKeyInitPassword 用户初始密码配置键。
	ConfigKeyInitPassword = "sys_user_initPassword"
	// ConfigKeyIndexSkinName 皮肤主题配置键。
	ConfigKeyIndexSkinName = "sys_index_skinName"
	// ConfigKeyIndexSideTheme 侧边栏主题配置键。
	ConfigKeyIndexSideTheme = "sys_index_sideTheme"

	// DefaultInitPassword 系统配置表缺失初始密码键时的回退值。
	DefaultInitPassword = "123456"
)
