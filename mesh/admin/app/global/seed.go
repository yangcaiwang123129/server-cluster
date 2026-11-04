package global

import (
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"server-cluster/share/consts"
	"server-cluster/share/enum"
	"server-cluster/share/model"
)

// seedAPI 种子接口：标题、路径、方法。
type seedAPI struct {
	Title  string
	Path   string
	Action string
}

// seedMenu 种子菜单及挂载的接口。
type seedMenu struct {
	Menu     model.SysMenu
	Apis     []seedAPI
	Children []seedMenu
}

// menuAPIRule 菜单-接口关系行（sys_menu_api_rule）。
type menuAPIRule struct {
	MenuID int `gorm:"column:menu_id"`
	APIID  int `gorm:"column:api_id"`
}

// pendingMenu 菜单树遍历中间态：parent 为已确定的父菜单 ID。
type pendingMenu struct {
	node   *seedMenu
	parent int
	id     int
}

// seed 首次启动时写入管理员、字典与系统配置；已有账号时视为已初始化。
//
// 菜单/接口/关系不在本函数内写入，统一交给启动流程中紧随其后的 syncMenus 处理，
// 避免同一份种子逻辑维护两套（逐条 vs 批量）。
func seed(gdb *gorm.DB) error {
	var count int64
	if err := gdb.Model(&model.SysUser{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return gdb.Transaction(func(tx *gorm.DB) error {
		if err := seedConfigs(tx); err != nil {
			return err
		}
		if err := seedDicts(tx); err != nil {
			return err
		}
		role := model.SysRole{RoleName: "超级管理员", RoleKey: consts.AdminRoleKey, RoleSort: 1, Status: consts.StatusEnabled, Remark: "拥有全部权限"}
		if err := tx.Create(&role).Error; err != nil {
			return err
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(consts.DefaultInitPassword), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		admin := model.SysUser{
			Username: "admin",
			Password: string(hash),
			NickName: "管理员",
			RoleId:   role.RoleId,
			Sex:      "1",
			Status:   consts.StatusEnabled,
			Remark:   "初始超级管理员",
		}
		return tx.Create(&admin).Error
	})
}

// seedConfigs 写入内置系统配置。
func seedConfigs(tx *gorm.DB) error {
	configs := []model.SysConfig{
		{ConfigName: "应用名称", ConfigKey: consts.ConfigKeyAppName, ConfigValue: "游戏运营后台", ConfigType: consts.FlagYes, IsFrontend: "1", Remark: "登录页与侧边栏标题"},
		{ConfigName: "应用Logo", ConfigKey: consts.ConfigKeyAppLogo, ConfigValue: "", ConfigType: consts.FlagYes, IsFrontend: "1", Remark: "留空使用内置图标"},
		{ConfigName: "用户初始密码", ConfigKey: consts.ConfigKeyInitPassword, ConfigValue: consts.DefaultInitPassword, ConfigType: consts.FlagYes, IsFrontend: "0", Remark: "新增用户未填密码时使用"},
		{ConfigName: "皮肤主题", ConfigKey: consts.ConfigKeyIndexSkinName, ConfigValue: "default", ConfigType: consts.FlagYes, IsFrontend: "0"},
		{ConfigName: "侧边栏主题", ConfigKey: consts.ConfigKeyIndexSideTheme, ConfigValue: "theme-dark", ConfigType: consts.FlagYes, IsFrontend: "0"},
	}
	return tx.Create(&configs).Error
}

// seedDicts 写入常用字典。
func seedDicts(tx *gorm.DB) error {
	types := []model.SysDictType{
		{DictName: "系统开关", DictType: "sys_normal_disable", Status: consts.StatusEnabled},
		{DictName: "显示状态", DictType: "sys_show_hide", Status: consts.StatusEnabled},
		{DictName: "用户性别", DictType: "sys_user_sex", Status: consts.StatusEnabled},
		{DictName: "系统是否", DictType: "sys_yes_no", Status: consts.StatusEnabled},
		{DictName: "通用状态", DictType: "sys_common_status", Status: consts.StatusEnabled},
		{DictName: "操作类型", DictType: "sys_oper_type", Status: consts.StatusEnabled},
	}
	if err := tx.Create(&types).Error; err != nil {
		return err
	}
	data := []model.SysDictData{
		{DictSort: 1, DictLabel: "正常", DictValue: "2", DictType: "sys_normal_disable", ListClass: "primary", IsDefault: consts.FlagYes, Status: consts.StatusEnabled},
		{DictSort: 2, DictLabel: "停用", DictValue: "1", DictType: "sys_normal_disable", ListClass: "danger", IsDefault: consts.FlagNo, Status: consts.StatusEnabled},
		{DictSort: 1, DictLabel: "显示", DictValue: "0", DictType: "sys_show_hide", ListClass: "primary", IsDefault: consts.FlagYes, Status: consts.StatusEnabled},
		{DictSort: 2, DictLabel: "隐藏", DictValue: "1", DictType: "sys_show_hide", ListClass: "danger", IsDefault: consts.FlagNo, Status: consts.StatusEnabled},
		{DictSort: 1, DictLabel: "男", DictValue: "1", DictType: "sys_user_sex", ListClass: "", IsDefault: consts.FlagYes, Status: consts.StatusEnabled},
		{DictSort: 2, DictLabel: "女", DictValue: "2", DictType: "sys_user_sex", ListClass: "", IsDefault: consts.FlagNo, Status: consts.StatusEnabled},
		{DictSort: 3, DictLabel: "未知", DictValue: "0", DictType: "sys_user_sex", ListClass: "", IsDefault: consts.FlagNo, Status: consts.StatusEnabled},
		{DictSort: 1, DictLabel: "是", DictValue: "Y", DictType: "sys_yes_no", ListClass: "primary", IsDefault: consts.FlagYes, Status: consts.StatusEnabled},
		{DictSort: 2, DictLabel: "否", DictValue: "N", DictType: "sys_yes_no", ListClass: "danger", IsDefault: consts.FlagNo, Status: consts.StatusEnabled},
		{DictSort: 1, DictLabel: "成功", DictValue: "2", DictType: "sys_common_status", ListClass: "primary", IsDefault: consts.FlagYes, Status: consts.StatusEnabled},
		{DictSort: 2, DictLabel: "失败", DictValue: "1", DictType: "sys_common_status", ListClass: "danger", IsDefault: consts.FlagNo, Status: consts.StatusEnabled},
		{DictSort: 1, DictLabel: "新增", DictValue: "1", DictType: "sys_oper_type", ListClass: "success", IsDefault: consts.FlagYes, Status: consts.StatusEnabled},
		{DictSort: 2, DictLabel: "修改", DictValue: "2", DictType: "sys_oper_type", ListClass: "primary", IsDefault: consts.FlagNo, Status: consts.StatusEnabled},
		{DictSort: 3, DictLabel: "删除", DictValue: "3", DictType: "sys_oper_type", ListClass: "danger", IsDefault: consts.FlagNo, Status: consts.StatusEnabled},
	}
	return tx.Create(&data).Error
}

// syncMenus 为已初始化的库补齐新增菜单与接口（按 menu_name 幂等，不覆盖既有菜单配置）。
func syncMenus(gdb *gorm.DB) error {
	return gdb.Transaction(func(tx *gorm.DB) error {
		return syncMenuNodes(tx, defaultMenus)
	})
}

// syncMenuNodes 批量补齐菜单树、接口与菜单-接口关系。
//
// 实现要点（把逐条 SELECT/INSERT 收敛为固定几条 SQL）：
//  1. 一次性预加载现有菜单、接口、关系三份字典；
//  2. 菜单按层级遍历：父 ID 确定后再处理子节点，同层新菜单合并为一次 CreateInBatches；
//  3. 新接口按 (path, action) 去重后一次批量插入；
//  4. 新关系按 (menu_id, api_id) 去重后一次批量插入；
//  5. 既有菜单仅在归属/路由/标题/排序/图标有差异时逐条更新（稳定库上为 0 条）。
func syncMenuNodes(tx *gorm.DB, roots []seedMenu) error {
	menusByName := make(map[string]model.SysMenu)
	var menus []model.SysMenu
	if err := tx.Find(&menus).Error; err != nil {
		return err
	}
	for _, m := range menus {
		menusByName[m.MenuName] = m
	}

	apisByKey := make(map[string]model.SysApi)
	var apis []model.SysApi
	if err := tx.Find(&apis).Error; err != nil {
		return err
	}
	for _, a := range apis {
		apisByKey[apiKey(a.Path, a.Action)] = a
	}

	rules := make(map[[2]int]struct{})
	var rels []menuAPIRule
	if err := tx.Table(enum.TableSysMenuApiRule.String()).
		Select("menu_id", "api_id").Find(&rels).Error; err != nil {
		return err
	}
	for _, r := range rels {
		rules[[2]int{r.MenuID, r.APIID}] = struct{}{}
	}

	// 1. 分层解析/创建菜单
	level := make([]pendingMenu, 0, len(roots))
	for i := range roots {
		level = append(level, pendingMenu{node: &roots[i]})
	}
	resolved := make([]pendingMenu, 0)
	for len(level) > 0 {
		var (
			toCreate  []model.SysMenu
			createIdx []int
		)
		for i := range level {
			pm := &level[i]
			want := pm.node.Menu
			if cur, ok := menusByName[want.MenuName]; ok {
				pm.id = cur.MenuId
				if updates := menuUpdates(cur, want, pm.parent); len(updates) > 0 {
					if err := tx.Model(&model.SysMenu{}).
						Where("menu_id = ?", cur.MenuId).Updates(updates).Error; err != nil {
						return err
					}
				}
				continue
			}
			created := want
			created.ParentId = pm.parent
			toCreate = append(toCreate, created)
			createIdx = append(createIdx, i)
		}
		if len(toCreate) > 0 {
			if err := tx.CreateInBatches(&toCreate, 100).Error; err != nil {
				return err
			}
			for j := range toCreate {
				level[createIdx[j]].id = toCreate[j].MenuId
				menusByName[toCreate[j].MenuName] = toCreate[j]
			}
		}
		resolved = append(resolved, level...)
		next := make([]pendingMenu, 0)
		for i := range level {
			for c := range level[i].node.Children {
				next = append(next, pendingMenu{node: &level[i].node.Children[c], parent: level[i].id})
			}
		}
		level = next
	}

	// 2. 批量创建新接口：(path, action) 幂等
	var newAPIs []model.SysApi
	newAPIKeys := make(map[string]struct{})
	for i := range resolved {
		for _, api := range resolved[i].node.Apis {
			k := apiKey(api.Path, api.Action)
			if _, ok := apisByKey[k]; ok {
				continue
			}
			if _, ok := newAPIKeys[k]; ok {
				continue
			}
			newAPIKeys[k] = struct{}{}
			newAPIs = append(newAPIs, model.SysApi{
				Path:   api.Path,
				Action: api.Action,
				Title:  api.Title,
				Type:   "BUS",
			})
		}
	}
	if len(newAPIs) > 0 {
		if err := tx.CreateInBatches(&newAPIs, 100).Error; err != nil {
			return err
		}
		for i := range newAPIs {
			apisByKey[apiKey(newAPIs[i].Path, newAPIs[i].Action)] = newAPIs[i]
		}
	}

	// 3. 批量创建菜单-接口关系：(menu_id, api_id) 幂等
	var newRules []menuAPIRule
	for i := range resolved {
		pm := &resolved[i]
		for _, api := range pm.node.Apis {
			record, ok := apisByKey[apiKey(api.Path, api.Action)]
			if !ok {
				continue
			}
			key := [2]int{pm.id, record.ApiId}
			if _, ok := rules[key]; ok {
				continue
			}
			rules[key] = struct{}{}
			newRules = append(newRules, menuAPIRule{MenuID: pm.id, APIID: record.ApiId})
		}
	}
	if len(newRules) > 0 {
		if err := tx.Table(enum.TableSysMenuApiRule.String()).
			CreateInBatches(&newRules, 100).Error; err != nil {
			return err
		}
	}
	return nil
}

// apiKey 接口幂等键：路径 + 方法。
func apiKey(path, action string) string { return path + "\x00" + action }

// menuUpdates 对比既有菜单与种子，只返回需要更新的字段。
func menuUpdates(cur, want model.SysMenu, parentID int) map[string]any {
	updates := make(map[string]any)
	if cur.ParentId != parentID {
		updates["parent_id"] = parentID
	}
	if cur.Path != want.Path {
		updates["path"] = want.Path
	}
	if cur.Title != want.Title {
		updates["title"] = want.Title
	}
	if cur.Sort != want.Sort {
		updates["sort"] = want.Sort
	}
	if want.Icon != "" && cur.Icon == "" {
		updates["icon"] = want.Icon
	}
	return updates
}
