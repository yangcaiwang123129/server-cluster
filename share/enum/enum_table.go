package enum

// Table 表名枚举
type Table string

const (
	TablePlayer          Table = "player"           // 玩家表（1:1）
	TablePlayerExtra     Table = "player_extra"     // 玩家拓展表（1:1）
	TablePlayerItem      Table = "player_item"      // 玩家物品表 持有（1:1，物品容器 JSON）
	TablePlayerEquip     Table = "player_equip"     // 玩家装备表 持有（1:1，装备 JSON）
	TablePlayerHero      Table = "player_hero"      // 玩家英雄表 持有（1:1，英雄 JSON）
	TablePlayerWeapon    Table = "player_weapon"    // 玩家武器表 持有（1:1，武器 JSON）
	TablePlayerSkill     Table = "player_skill"     // 玩家技能表 持有（1:1，技能 JSON）
)

// 游戏打点记录表
const (
	TableItemRecord   Table = "item_record"   // 物品打点
	TableEquipRecord  Table = "equip_record"  // 装备打点
	TableWeaponRecord Table = "weapon_record" // 武器打点
	TableSkillRecord  Table = "skill_record"  // 技能打点
	TableHeroRecord   Table = "hero_record"   // 英雄打点
)

// 管理后台表
const (
	TableSysUser        Table = "sys_user"          // 后台账号
	TableSysRole        Table = "sys_role"          // 角色
	TableSysMenu        Table = "sys_menu"          // 菜单/按钮权限
	TableSysApi         Table = "sys_api"           // 后台接口登记
	TableSysDictType    Table = "sys_dict_type"     // 字典类型
	TableSysDictData    Table = "sys_dict_data"     // 字典项
	TableSysConfig      Table = "sys_config"        // 系统配置
	TableSysLoginLog    Table = "sys_login_log"     // 登录审计
	TableSysOperaLog    Table = "sys_opera_log"     // 操作审计
	TableSysCasbinRule  Table = "sys_casbin_rule"   // 接口放行规则
	TableSysMenuApiRule Table = "sys_menu_api_rule" // 菜单与接口关联
	TableClientVersion  Table = "client_version"    // 客户端版本
)

// String 返回表名字符串。
func (t Table) String() string { return string(t) }
