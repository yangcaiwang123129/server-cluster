package global

import (
	"server-cluster/share/model"
)

// defaultMenus 初始菜单树：只包含管理后台保留的业务模块。
var defaultMenus = []seedMenu{
	{
		Menu: model.SysMenu{MenuName: "Admin", Title: "系统管理", Path: "/admin", MenuType: "M", Component: "Layout", Icon: "system", Sort: 1, Visible: "0", IsFrame: "1"},
		Children: []seedMenu{
			{
				Menu: model.SysMenu{MenuName: "SysUserManage", Title: "用户管理", Path: "/admin/sys-user/index", Component: "/admin/sys-user/index", MenuType: "C", Icon: "user", Permission: "sys:user:list", Sort: 1, Visible: "0", IsFrame: "1"},
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
				Menu: model.SysMenu{MenuName: "SysRoleManage", Title: "角色管理", Path: "/admin/sys-role/index", Component: "/admin/sys-role/index", MenuType: "C", Icon: "peoples", Permission: "sys:role:list", Sort: 2, Visible: "0", IsFrame: "1"},
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
				Menu: model.SysMenu{MenuName: "SysMenuManage", Title: "菜单管理", Path: "/admin/sys-menu/index", Component: "/admin/sys-menu/index", MenuType: "C", Icon: "nested", Permission: "sys:menu:list", Sort: 3, Visible: "0", IsFrame: "1"},
				Apis: []seedAPI{
					{"菜单列表", "/api/v1/menu", "GET"},
					{"菜单详情", "/api/v1/menu/:menuId", "GET"},
					{"新增菜单", "/api/v1/menu", "POST"},
					{"修改菜单", "/api/v1/menu/:id", "PUT"},
					{"删除菜单", "/api/v1/menu", "DELETE"},
				},
			},
			{
				Menu: model.SysMenu{MenuName: "SysApiManage", Title: "接口管理", Path: "/admin/sys-api/index", Component: "/admin/sys-api/index", MenuType: "C", Icon: "api-server", Permission: "sys:api:list", Sort: 4, Visible: "0", IsFrame: "1"},
				Apis: []seedAPI{
					{"接口列表", "/api/v1/sys-api", "GET"},
					{"接口详情", "/api/v1/sys-api/:id", "GET"},
					{"新增接口", "/api/v1/sys-api", "POST"},
					{"修改接口", "/api/v1/sys-api/:id", "PUT"},
					{"删除接口", "/api/v1/sys-api", "DELETE"},
				},
			},
			{
				Menu: model.SysMenu{MenuName: "Dict", Title: "字典管理", Path: "/admin/dict/index", Component: "/admin/dict/index", MenuType: "C", Icon: "dict", Permission: "sys:dict:list", Sort: 5, Visible: "0", IsFrame: "1"},
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
						Menu: model.SysMenu{MenuName: "SysDictDataManage", Title: "字典数据", Path: "/admin/dict/data/:dictId", Component: "/admin/dict/data", MenuType: "C", Icon: "list", Permission: "sys:dict:data", Sort: 1, Visible: "1", IsFrame: "1"},
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
				Menu: model.SysMenu{MenuName: "SysConfigManage", Title: "参数管理", Path: "/admin/sys-config/index", Component: "/admin/sys-config/index", MenuType: "C", Icon: "base-info", Permission: "sys:config:list", Sort: 6, Visible: "0", IsFrame: "1"},
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
				Menu: model.SysMenu{MenuName: "SysConfigSet", Title: "参数设置", Path: "/admin/sys-config/set", Component: "/admin/sys-config/set", MenuType: "C", Icon: "info-setting", Permission: "sys:config:set", Sort: 7, Visible: "0", IsFrame: "1"},
				Apis: []seedAPI{
					{"读取参数设置", "/api/v1/set-config", "GET"},
					{"保存参数设置", "/api/v1/set-config", "PUT"},
				},
			},
		},
	},
	{
		Menu: model.SysMenu{MenuName: "Log", Title: "日志管理", Path: "/log", MenuType: "M", Component: "Layout", Icon: "log", Sort: 2, Visible: "0", IsFrame: "1"},
		Children: []seedMenu{
			{
				Menu: model.SysMenu{MenuName: "SysLoginLogManage", Title: "登录日志", Path: "/log/sys-login-log/index", Component: "/admin/sys-login-log/index", MenuType: "C", Icon: "logininfor", Permission: "sys:login-log:list", Sort: 1, Visible: "0", IsFrame: "1"},
				Apis: []seedAPI{
					{"登录日志列表", "/api/v1/sys-login-log", "GET"},
					{"登录日志详情", "/api/v1/sys-login-log/:id", "GET"},
					{"删除登录日志", "/api/v1/sys-login-log", "DELETE"},
				},
			},
			{
				Menu: model.SysMenu{MenuName: "OperLog", Title: "操作日志", Path: "/log/sys-oper-log/index", Component: "/admin/sys-oper-log/index", MenuType: "C", Icon: "err-list", Permission: "sys:opera-log:list", Sort: 2, Visible: "0", IsFrame: "1"},
				Apis: []seedAPI{
					{"操作日志列表", "/api/v1/sys-opera-log", "GET"},
					{"删除操作日志", "/api/v1/sys-opera-log", "DELETE"},
					{"清空操作日志", "/api/v1/operlog/clean", "DELETE"},
				},
			},
		},
	},
	{
		Menu: model.SysMenu{MenuName: "Game", Title: "游戏管理", Path: "/game", MenuType: "M", Component: "Layout", Icon: "star", Sort: 4, Visible: "0", IsFrame: "1"},
		Children: []seedMenu{
			{
				Menu: model.SysMenu{MenuName: "PlayerManage", Title: "玩家列表", Path: "/game/player/index", Component: "/admin/player/index", MenuType: "C", Icon: "user-info", Permission: "game:player:list", Sort: 1, Visible: "0", IsFrame: "1"},
				Apis: []seedAPI{
					{"玩家列表", "/api/v1/gm/player", "GET"},
					{"在线玩家", "/api/v1/gm/player/online", "GET"},
					{"踢玩家下线", "/api/v1/gm/player/kick", "POST"},
				},
			},
			{
				Menu: model.SysMenu{MenuName: "GameContainerManage", Title: "容器管理", Path: "/game/container/index", Component: "/admin/game-container/index", MenuType: "C", Icon: "component", Permission: "game:gm:container:list", Sort: 2, Visible: "0", IsFrame: "1"},
				Apis: []seedAPI{
					{"物品配置列表", "/api/v1/gm/catalog", "GET"},
					{"GM容器列表", "/api/v1/gm/container", "GET"},
					{"GM添加物品", "/api/v1/gm/container/item", "POST"},
					{"GM移动物品", "/api/v1/gm/container/move", "POST"},
					{"GM旋转物品", "/api/v1/gm/container/rotate", "POST"},
					{"GM整理容器", "/api/v1/gm/container/sort", "POST"},
					{"GM转移物品", "/api/v1/gm/container/transfer", "POST"},
				},
			},
			{
				Menu: model.SysMenu{MenuName: "GameHeroManage", Title: "英雄管理", Path: "/game/hero/index", Component: "/admin/game-hero/index", MenuType: "C", Icon: "vip", Permission: "game:gm:hero:list", Sort: 4, Visible: "0", IsFrame: "1"},
				Apis: []seedAPI{
					{"游戏配置目录", "/api/v1/gm/catalog", "GET"},
					{"GM英雄列表", "/api/v1/gm/hero", "GET"},
					{"GM新增英雄", "/api/v1/gm/hero", "POST"},
					{"GM更新英雄", "/api/v1/gm/hero", "PUT"},
				},
			},
			{
				Menu: model.SysMenu{MenuName: "GameWeaponManage", Title: "武器管理", Path: "/game/weapon/index", Component: "/admin/game-weapon/index", MenuType: "C", Icon: "tool", Permission: "game:gm:weapon:list", Sort: 5, Visible: "0", IsFrame: "1"},
				Apis: []seedAPI{
					{"游戏配置目录", "/api/v1/gm/catalog", "GET"},
					{"GM武器列表", "/api/v1/gm/weapon", "GET"},
					{"GM新增武器", "/api/v1/gm/weapon", "POST"},
					{"GM更新武器", "/api/v1/gm/weapon", "PUT"},
				},
			},
			{
				Menu: model.SysMenu{MenuName: "GameSkillManage", Title: "技能管理", Path: "/game/skill/index", Component: "/admin/game-skill/index", MenuType: "C", Icon: "skill", Permission: "game:gm:skill:list", Sort: 6, Visible: "0", IsFrame: "1"},
				Apis: []seedAPI{
					{"游戏配置目录", "/api/v1/gm/catalog", "GET"},
					{"GM技能列表", "/api/v1/gm/skill", "GET"},
					{"GM新增技能", "/api/v1/gm/skill", "POST"},
					{"GM更新技能", "/api/v1/gm/skill", "PUT"},
				},
			},
			{
				Menu: model.SysMenu{MenuName: "GameEquipManage", Title: "装备管理", Path: "/game/equip/index", Component: "/admin/game-equip/index", MenuType: "C", Icon: "component", Permission: "game:gm:equip:list", Sort: 7, Visible: "0", IsFrame: "1"},
				Apis: []seedAPI{
					{"游戏配置目录", "/api/v1/gm/catalog", "GET"},
					{"GM装备列表", "/api/v1/gm/equip", "GET"},
					{"GM新增装备", "/api/v1/gm/equip", "POST"},
					{"GM更新装备", "/api/v1/gm/equip", "PUT"},
				},
			},
		},
	},
	{
		Menu: model.SysMenu{MenuName: "Client", Title: "客户端管理", Path: "/client", MenuType: "M", Component: "Layout", Icon: "app-group-fill", Sort: 3, Visible: "0", IsFrame: "1"},
		Children: []seedMenu{
			{
				Menu: model.SysMenu{MenuName: "ClientSimulator", Title: "模拟器", Path: "/client/simulator/index", Component: "/admin/simulator/index", Icon: "scene", MenuType: "C", Permission: "game:sim:list", Sort: 1, Visible: "0", IsFrame: "1"},
				Apis: []seedAPI{
					{"模拟器默认参数", "/api/v1/gm/client/sim/options", "GET"},
					{"模拟器登录", "/api/v1/gm/client/sim/login", "POST"},
					{"模拟器连接网关", "/api/v1/gm/client/sim/connect", "POST"},
					{"模拟器流程步骤", "/api/v1/gm/client/sim/step", "POST"},
					{"模拟器关闭会话", "/api/v1/gm/client/sim/close", "POST"},
				},
			},
			{
				Menu: model.SysMenu{MenuName: "ClientVersionManage", Title: "版本管理", Path: "/client/version/index", Component: "/admin/client-version/index", Icon: "build", MenuType: "C", Permission: "client:version:list", Sort: 2, Visible: "0", IsFrame: "1"},
				Apis: []seedAPI{
					{"客户端版本列表", "/api/v1/gm/client/version", "GET"},
					{"客户端版本详情", "/api/v1/gm/client/version/:id", "GET"},
					{"新增客户端版本", "/api/v1/gm/client/version", "POST"},
					{"修改客户端版本", "/api/v1/gm/client/version/:id", "PUT"},
					{"删除客户端版本", "/api/v1/gm/client/version", "DELETE"},
					{"客户端版本启停", "/api/v1/gm/client/version-status", "PUT"},
				},
			},
		},
	},
	{
		Menu: model.SysMenu{MenuName: "GameRecord", Title: "游戏记录", Path: "/record", MenuType: "M", Component: "Layout", Icon: "form", Sort: 5, Visible: "0", IsFrame: "1"},
		Children: []seedMenu{
			{
				Menu: model.SysMenu{MenuName: "GameRecordGain", Title: "获得记录", Path: "/record/gain", Component: "/admin/game-record/gain", MenuType: "C", Icon: "list", Permission: "game:gm:record:gain", Sort: 1, Visible: "0", IsFrame: "1"},
				Apis: []seedAPI{
					{"获得记录列表", "/api/v1/gm/record", "GET"},
				},
			},
			{
				Menu: model.SysMenu{MenuName: "GameRecordConsume", Title: "消耗记录", Path: "/record/consume", Component: "/admin/game-record/consume", MenuType: "C", Icon: "err-list", Permission: "game:gm:record:consume", Sort: 2, Visible: "0", IsFrame: "1"},
				Apis: []seedAPI{
					{"消耗记录列表", "/api/v1/gm/record", "GET"},
				},
			},
		},
	},
}
