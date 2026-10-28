package global

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"server-cluster/share/consts"
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

// seed 首次启动时写入管理员、菜单、字典与系统配置；已有账号时视为已初始化。
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
		hash, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
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
		if err := tx.Create(&admin).Error; err != nil {
			return err
		}
		return seedMenus(tx, defaultMenus, 0)
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
		{DictName: "系统开关", DictType: "sys_normal_disable", Status: "2"},
		{DictName: "显示状态", DictType: "sys_show_hide", Status: "2"},
		{DictName: "用户性别", DictType: "sys_user_sex", Status: "2"},
		{DictName: "系统是否", DictType: "sys_yes_no", Status: "2"},
		{DictName: "通用状态", DictType: "sys_common_status", Status: "2"},
		{DictName: "操作类型", DictType: "sys_oper_type", Status: "2"},
	}
	if err := tx.Create(&types).Error; err != nil {
		return err
	}
	data := []model.SysDictData{
		{DictSort: 1, DictLabel: "正常", DictValue: "2", DictType: "sys_normal_disable", ListClass: "primary", IsDefault: "Y", Status: "2"},
		{DictSort: 2, DictLabel: "停用", DictValue: "1", DictType: "sys_normal_disable", ListClass: "danger", IsDefault: "N", Status: "2"},
		{DictSort: 1, DictLabel: "显示", DictValue: "0", DictType: "sys_show_hide", ListClass: "primary", IsDefault: "Y", Status: "2"},
		{DictSort: 2, DictLabel: "隐藏", DictValue: "1", DictType: "sys_show_hide", ListClass: "danger", IsDefault: "N", Status: "2"},
		{DictSort: 1, DictLabel: "男", DictValue: "1", DictType: "sys_user_sex", ListClass: "", IsDefault: "Y", Status: "2"},
		{DictSort: 2, DictLabel: "女", DictValue: "2", DictType: "sys_user_sex", ListClass: "", IsDefault: "N", Status: "2"},
		{DictSort: 3, DictLabel: "未知", DictValue: "0", DictType: "sys_user_sex", ListClass: "", IsDefault: "N", Status: "2"},
		{DictSort: 1, DictLabel: "是", DictValue: "Y", DictType: "sys_yes_no", ListClass: "primary", IsDefault: "Y", Status: "2"},
		{DictSort: 2, DictLabel: "否", DictValue: "N", DictType: "sys_yes_no", ListClass: "danger", IsDefault: "N", Status: "2"},
		{DictSort: 1, DictLabel: "成功", DictValue: "2", DictType: "sys_common_status", ListClass: "primary", IsDefault: "Y", Status: "2"},
		{DictSort: 2, DictLabel: "失败", DictValue: "1", DictType: "sys_common_status", ListClass: "danger", IsDefault: "N", Status: "2"},
		{DictSort: 1, DictLabel: "新增", DictValue: "1", DictType: "sys_oper_type", ListClass: "success", IsDefault: "Y", Status: "2"},
		{DictSort: 2, DictLabel: "修改", DictValue: "2", DictType: "sys_oper_type", ListClass: "primary", IsDefault: "N", Status: "2"},
		{DictSort: 3, DictLabel: "删除", DictValue: "3", DictType: "sys_oper_type", ListClass: "danger", IsDefault: "N", Status: "2"},
	}
	return tx.Create(&data).Error
}

// seedMenus 递归写入菜单与接口关系。
func seedMenus(tx *gorm.DB, items []seedMenu, parentId int) error {
	for i := range items {
		m := items[i].Menu
		m.ParentId = parentId
		if err := tx.Create(&m).Error; err != nil {
			return err
		}
		for _, api := range items[i].Apis {
			var record model.SysApi
			err := tx.Where("path = ? AND action = ?", api.Path, api.Action).First(&record).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				record = model.SysApi{Path: api.Path, Action: api.Action, Title: api.Title, Type: "BUS"}
				if err := tx.Create(&record).Error; err != nil {
					return err
				}
			} else if err != nil {
				return err
			}
			if err := tx.Table("sys_menu_api_rule").
				Create(map[string]any{"menu_id": m.MenuId, "api_id": record.ApiId}).Error; err != nil {
				return err
			}
		}
		if err := seedMenus(tx, items[i].Children, m.MenuId); err != nil {
			return err
		}
	}
	return nil
}

// syncMenus 为已初始化的库补齐新增菜单与接口（按 menu_name 幂等，不覆盖既有菜单配置）。
func syncMenus(gdb *gorm.DB) error {
	return gdb.Transaction(func(tx *gorm.DB) error {
		return syncMenuNodes(tx, defaultMenus, 0)
	})
}

// syncMenuNodes 递归补齐菜单节点：已存在的菜单沿用原配置与 ID，只补缺失的接口关系。
func syncMenuNodes(tx *gorm.DB, items []seedMenu, parentId int) error {
	for i := range items {
		var menu model.SysMenu
		err := tx.Where("menu_name = ?", items[i].Menu.MenuName).First(&menu).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			menu = items[i].Menu
			menu.ParentId = parentId
			if err := tx.Create(&menu).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}

		for _, api := range items[i].Apis {
			var record model.SysApi
			err := tx.Where("path = ? AND action = ?", api.Path, api.Action).First(&record).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				record = model.SysApi{Path: api.Path, Action: api.Action, Title: api.Title, Type: "BUS"}
				if err := tx.Create(&record).Error; err != nil {
					return err
				}
			} else if err != nil {
				return err
			}

			var count int64
			if err := tx.Table("sys_menu_api_rule").
				Where("menu_id = ? AND api_id = ?", menu.MenuId, record.ApiId).
				Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				if err := tx.Table("sys_menu_api_rule").
					Create(map[string]any{"menu_id": menu.MenuId, "api_id": record.ApiId}).Error; err != nil {
					return err
				}
			}
		}

		if err := syncMenuNodes(tx, items[i].Children, menu.MenuId); err != nil {
			return err
		}
	}
	return nil
}
