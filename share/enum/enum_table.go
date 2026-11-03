package enum

// Table 表名枚举
type Table string

const (
	TableActivity       Table = "activity"        // 全局活动表
	TablePlayer         Table = "player"          // 玩家表（1:1）
	TablePlayerExtra    Table = "player_extra"    // 玩家拓展表（1:1）
	TablePlayerAuth     Table = "player_auth"     // 玩家认证表
	TablePlayerItem     Table = "player_item"     // 玩家物品表
	TablePlayerEquip    Table = "player_equip"    // 玩家装备表
	TablePlayerHero     Table = "player_hero"     // 玩家英雄表
	TablePlayerWeapon   Table = "player_weapon"   // 玩家武器表
	TablePlayerSkill    Table = "player_skill"    // 玩家技能表
	TablePlayerMail     Table = "player_mail"     // 玩家邮件表
	TablePlayerActivity Table = "player_activity" // 玩家活动表
	TablePlayerTask     Table = "player_task"     // 玩家任务表
)

// 游戏打点记录表
const (
	TableRecordItem   Table = "record_item"   // 打点物品记录
	TableRecordEquip  Table = "record_equip"  // 打点装备记录
	TableRecordWeapon Table = "record_weapon" // 打点武器记录
	TableRecordSkill  Table = "record_skill"  // 打点技能记录
	TableRecordHero   Table = "record_hero"   // 打点英雄记录
)

// 管理后台表
const (
	TableSysUser        Table = "sys_user"          // 后台账号
	TableSysRole        Table = "sys_role"          // 后台角色
	TableSysMenu        Table = "sys_menu"          // 后台菜单/按钮权限
	TableSysApi         Table = "sys_api"           // 后台接口登记
	TableSysDictType    Table = "sys_dict_type"     // 后台字典类型
	TableSysDictData    Table = "sys_dict_data"     // 后台字典项
	TableSysConfig      Table = "sys_config"        // 后台系统配置
	TableSysLoginLog    Table = "sys_login_log"     // 后台登录审计
	TableSysOperaLog    Table = "sys_opera_log"     // 后台操作审计
	TableSysCasbinRule  Table = "sys_casbin_rule"   // 后台接口放行规则
	TableSysMenuApiRule Table = "sys_menu_api_rule" // 后台菜单与接口关联
)

const (
	TableClient  Table = "client"  // 客户端版本
	TableCluster Table = "cluster" // 集群节点表
)

// String 返回表名字符串。
func (t Table) String() string { return string(t) }
