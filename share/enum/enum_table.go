package enum

// Table 表名枚举：与 share/model 各表模型的 TableName() 一一对应，新增表时先在此登记。
type Table string

const (
	TableAccount        Table = "account"         // 账号表
	TablePlayer         Table = "player"          // 玩家表
	TablePlayerCore     Table = "player_core"     // 玩家核心数据表
	TablePlayerAssets   Table = "player_assets"   // 玩家资产表
	TablePlayerProfile  Table = "player_profile"  // 玩家档案表
	TablePlayerBackpack Table = "player_backpack" // 玩家背包表（data 列整包 JSON）
)

// 管理后台表（mesh/admin）。
const (
	TableSysUser     Table = "sys_user"      // 后台账号
	TableSysRole     Table = "sys_role"      // 角色
	TableSysMenu     Table = "sys_menu"      // 菜单/按钮权限
	TableSysApi      Table = "sys_api"       // 后台接口登记
	TableSysDictType Table = "sys_dict_type" // 字典类型
	TableSysDictData Table = "sys_dict_data" // 字典项
	TableSysConfig   Table = "sys_config"    // 系统配置
	TableSysLoginLog Table = "sys_login_log" // 登录审计
	TableSysOperaLog Table = "sys_opera_log" // 操作审计
	TableCasbinRule  Table = "casbin_rule"   // 接口放行规则
)

// String 返回表名字符串。
func (t Table) String() string { return string(t) }
