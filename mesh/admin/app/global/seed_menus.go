package global

import (
	"server-cluster/share/model"
)

// defaultMenus 初始菜单树：只包含管理后台保留的业务模块。
var defaultMenus = []seedMenu{
	{
		Menu: model.SysMenu{MenuName: "Admin", Title: "系统管理", Path: "/admin", MenuType: "M", Component: "Layout", Sort: 1, Visible: "0", IsFrame: "1"},
		Children: []seedMenu{
			{
				Menu: model.SysMenu{MenuName: "SysUserManage", Title: "用户管理", Path: "/admin/sys-user/index", Component: "/admin/sys-user/index", MenuType: "C", Permission: "sys:user:list", Sort: 1, Visible: "0", IsFrame: "1"},
				Apis: []seedAPI{
					{"用户列表", "/api/v1/sys-user", "GET"},
					{"用户详情", "/api/v1/sys-user/:userId", "GET"},
					{"新增用户", "/api/v1/sys-user", "POST"},
					{"修改用户", "/api/v1/sys-user", "PUT"},
					{"删除用户", "/api/v1/sys-user", "DELETE"},
					{"用户状态", "/api/v1/user/status", "PUT"},
					{"重置密码", "/api/v1/user/pwd/reset", "PUT"},
				},
			},
			{
				Menu: model.SysMenu{MenuName: "SysRoleManage", Title: "角色管理", Path: "/admin/sys-role/index", Component: "/admin/sys-role/index", MenuType: "C", Permission: "sys:role:list", Sort: 2, Visible: "0", IsFrame: "1"},
				Apis: []seedAPI{
					{"角色列表", "/api/v1/role", "GET"},
					{"角色详情", "/api/v1/role/:roleId", "GET"},
					{"新增角色", "/api/v1/role", "POST"},
					{"修改角色", "/api/v1/role/:roleId", "PUT"},
					{"删除角色", "/api/v1/role", "DELETE"},
					{"角色状态", "/api/v1/role-status", "PUT"},
					{"角色菜单树", "/api/v1/roleMenuTreeselect/:roleId", "GET"},
				},
			},
			{
				Menu: model.SysMenu{MenuName: "SysMenuManage", Title: "菜单管理", Path: "/admin/sys-menu/index", Component: "/admin/sys-menu/index", MenuType: "C", Permission: "sys:menu:list", Sort: 3, Visible: "0", IsFrame: "1"},
				Apis: []seedAPI{
					{"菜单列表", "/api/v1/menu", "GET"},
					{"菜单详情", "/api/v1/menu/:menuId", "GET"},
					{"新增菜单", "/api/v1/menu", "POST"},
					{"修改菜单", "/api/v1/menu/:id", "PUT"},
					{"删除菜单", "/api/v1/menu", "DELETE"},
				},
			},
			{
				Menu: model.SysMenu{MenuName: "SysApiManage", Title: "接口管理", Path: "/admin/sys-api/index", Component: "/admin/sys-api/index", MenuType: "C", Permission: "sys:api:list", Sort: 4, Visible: "0", IsFrame: "1"},
				Apis: []seedAPI{
					{"接口列表", "/api/v1/sys-api", "GET"},
					{"接口详情", "/api/v1/sys-api/:id", "GET"},
					{"新增接口", "/api/v1/sys-api", "POST"},
					{"修改接口", "/api/v1/sys-api/:id", "PUT"},
					{"删除接口", "/api/v1/sys-api", "DELETE"},
				},
			},
			{
				Menu: model.SysMenu{MenuName: "Dict", Title: "字典管理", Path: "/admin/dict/index", Component: "/admin/dict/index", MenuType: "C", Permission: "sys:dict:list", Sort: 5, Visible: "0", IsFrame: "1"},
				Apis: []seedAPI{
					{"字典类型列表", "/api/v1/dict/type", "GET"},
					{"字典类型详情", "/api/v1/dict/type/:id", "GET"},
					{"新增字典类型", "/api/v1/dict/type", "POST"},
					{"修改字典类型", "/api/v1/dict/type/:id", "PUT"},
					{"删除字典类型", "/api/v1/dict/type", "DELETE"},
					{"字典类型下拉", "/api/v1/dict/type-option-select", "GET"},
				},
				Children: []seedMenu{
					{
						Menu: model.SysMenu{MenuName: "SysDictDataManage", Title: "字典数据", Path: "/admin/dict/data/:dictId", Component: "/admin/dict/data", MenuType: "C", Permission: "sys:dict:data", Sort: 1, Visible: "1", IsFrame: "1"},
						Apis: []seedAPI{
							{"字典数据列表", "/api/v1/dict/data", "GET"},
							{"字典数据详情", "/api/v1/dict/data/:dictCode", "GET"},
							{"新增字典数据", "/api/v1/dict/data", "POST"},
							{"修改字典数据", "/api/v1/dict/data/:dictCode", "PUT"},
							{"删除字典数据", "/api/v1/dict/data", "DELETE"},
							{"字典数据下拉", "/api/v1/dict-data/option-select", "GET"},
						},
					},
				},
			},
			{
				Menu: model.SysMenu{MenuName: "SysConfigManage", Title: "参数管理", Path: "/admin/sys-config/index", Component: "/admin/sys-config/index", MenuType: "C", Permission: "sys:config:list", Sort: 6, Visible: "0", IsFrame: "1"},
				Apis: []seedAPI{
					{"参数列表", "/api/v1/config", "GET"},
					{"参数详情", "/api/v1/config/:id", "GET"},
					{"按键取值", "/api/v1/configKey/:key", "GET"},
					{"新增参数", "/api/v1/config", "POST"},
					{"修改参数", "/api/v1/config/:id", "PUT"},
					{"删除参数", "/api/v1/config", "DELETE"},
				},
			},
			{
				Menu: model.SysMenu{MenuName: "SysConfigSet", Title: "参数设置", Path: "/admin/sys-config/set", Component: "/admin/sys-config/set", MenuType: "C", Permission: "sys:config:set", Sort: 7, Visible: "0", IsFrame: "1"},
				Apis: []seedAPI{
					{"读取参数设置", "/api/v1/set-config", "GET"},
					{"保存参数设置", "/api/v1/set-config", "PUT"},
				},
			},
		},
	},
	{
		Menu: model.SysMenu{MenuName: "Player", Title: "玩家管理", Path: "/player", MenuType: "M", Component: "Layout", Sort: 3, Visible: "0", IsFrame: "1"},
		Children: []seedMenu{
			{
				Menu: model.SysMenu{MenuName: "PlayerManage", Title: "玩家列表", Path: "/player/player/index", Component: "/admin/player/index", MenuType: "C", Permission: "game:player:list", Sort: 1, Visible: "0", IsFrame: "1"},
				Apis: []seedAPI{
					{"玩家列表", "/api/v1/player", "GET"},
					{"在线玩家", "/api/v1/player/online", "GET"},
					{"踢玩家下线", "/api/v1/player/kick", "POST"},
				},
			},
			{
				Menu: model.SysMenu{MenuName: "PlayerItemManage", Title: "玩家物品", Path: "/player/item/index", Component: "/admin/player-item/index", MenuType: "C", Permission: "game:player:item:list", Sort: 2, Visible: "0", IsFrame: "1"},
				Apis: []seedAPI{
					{"物品配置列表", "/api/v1/item", "GET"},
					{"玩家容器详情", "/api/v1/player/container", "GET"},
					{"添加物品", "/api/v1/player/container/item", "POST"},
					{"移动物品", "/api/v1/player/container/item", "PUT"},
					{"整理物品", "/api/v1/player/container/sort", "POST"},
				},
			},
		},
	},
	{
		Menu: model.SysMenu{MenuName: "Log", Title: "日志管理", Path: "/log", MenuType: "M", Component: "Layout", Sort: 2, Visible: "0", IsFrame: "1"},
		Children: []seedMenu{
			{
				Menu: model.SysMenu{MenuName: "SysLoginLogManage", Title: "登录日志", Path: "/log/sys-login-log/index", Component: "/admin/sys-login-log/index", MenuType: "C", Permission: "sys:login-log:list", Sort: 1, Visible: "0", IsFrame: "1"},
				Apis: []seedAPI{
					{"登录日志列表", "/api/v1/sys-login-log", "GET"},
					{"登录日志详情", "/api/v1/sys-login-log/:id", "GET"},
					{"删除登录日志", "/api/v1/sys-login-log", "DELETE"},
				},
			},
			{
				Menu: model.SysMenu{MenuName: "OperLog", Title: "操作日志", Path: "/log/sys-oper-log/index", Component: "/admin/sys-oper-log/index", MenuType: "C", Permission: "sys:opera-log:list", Sort: 2, Visible: "0", IsFrame: "1"},
				Apis: []seedAPI{
					{"操作日志列表", "/api/v1/sys-opera-log", "GET"},
					{"删除操作日志", "/api/v1/sys-opera-log", "DELETE"},
					{"清空操作日志", "/api/v1/operlog/clean", "DELETE"},
				},
			},
		},
	},
}
