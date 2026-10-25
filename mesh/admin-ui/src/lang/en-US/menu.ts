/**
 * Menu titles, keyed by `sys_menu.menu_name`.
 *
 * Mirrors what mesh/admin/app/seed_menus.go creates: the directory menus plus
 * the page menus carried by every kept module. Buttons and untranslated menus
 * the user creates have no entry here, and that is a normal state, not a gap --
 * lang/backend.ts falls back to the database title.
 *
 * The comment on each line is the Chinese the database actually stores, so a
 * reviewer can check a translation without opening the seed file.
 */
export default {
  Admin: 'System', // 系统管理
  SysUserManage: 'User Management', // 用户管理
  SysRoleManage: 'Role Management', // 角色管理
  SysMenuManage: 'Menu Management', // 菜单管理
  SysApiManage: 'API Management', // 接口管理
  Dict: 'Dictionary', // 字典管理
  SysDictDataManage: 'Dictionary Data', // 字典数据
  SysConfigManage: 'Parameters', // 参数管理
  SysConfigSet: 'Settings', // 参数设置
  Log: 'Logs', // 日志管理
  SysLoginLogManage: 'Login Logs', // 登录日志
  OperLog: 'Operation Logs' // 操作日志
}
