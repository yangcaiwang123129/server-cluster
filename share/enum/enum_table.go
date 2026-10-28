package enum

// Table 表名枚举：与 share/model 各表模型的 TableName() 一一对应。
// 命名规范：全部单数、统一 player_ 前缀；1:1 表主键 player_id，1:N 表主键 (player_id, 业务ID)。
type Table string

const (
	TablePlayerBase      Table = "player_base"      // 玩家账号（1:1）
	TablePlayerCore      Table = "player_core"      // 玩家核心数值（1:1）
	TablePlayerProfile   Table = "player_profile"   // 玩家展示资料（1:1）
	TablePlayerAsset     Table = "player_asset"     // 玩家资产（1:N，player_id + asset_type）
	TablePlayerHero      Table = "player_hero"      // 玩家英雄持有（1:1，英雄数组 JSON）
	TablePlayerSkill     Table = "player_skill"     // 玩家技能持有（1:1，技能数组 JSON）
	TablePlayerWeapon    Table = "player_weapon"    // 玩家武器持有（1:1，武器数组 JSON）
	TablePlayerGear      Table = "player_gear"      // 玩家装备持有（1:1，装备数组 JSON）
	TablePlayerContainer Table = "player_container" // 玩家容器（1:1）
)

// 管理后台表（mesh/admin）。
const (
	TableSysUser       Table = "sys_user"        // 后台账号
	TableSysRole       Table = "sys_role"        // 角色
	TableSysMenu       Table = "sys_menu"        // 菜单/按钮权限
	TableSysApi        Table = "sys_api"         // 后台接口登记
	TableSysDictType   Table = "sys_dict_type"   // 字典类型
	TableSysDictData   Table = "sys_dict_data"   // 字典项
	TableSysConfig     Table = "sys_config"      // 系统配置
	TableSysLoginLog   Table = "sys_login_log"   // 登录审计
	TableSysOperaLog   Table = "sys_opera_log"   // 操作审计
	TableSysCasbinRule Table = "sys_casbin_rule" // 接口放行规则
)

// String 返回表名字符串。
func (t Table) String() string { return string(t) }
