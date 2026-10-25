package repo

import (
	"sort"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"server-cluster/share/model"
)

// ApiQuery 接口列表过滤条件。
type ApiQuery struct {
	Title  string
	Path   string
	Action string
	Type   string
	Order  string
}

// ListApis 分页查询接口登记。
func (r *Repo) ListApis(q ApiQuery, pageIndex, pageSize int) ([]model.SysApi, int64, error) {
	tx := r.db.Model(&model.SysApi{})
	if q.Title != "" {
		tx = tx.Where("title LIKE ?", like(q.Title))
	}
	if q.Path != "" {
		tx = tx.Where("path LIKE ?", like(q.Path))
	}
	if q.Action != "" {
		tx = tx.Where("action = ?", q.Action)
	}
	if q.Type != "" {
		tx = tx.Where("type = ?", q.Type)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset, limit := normalizePage(pageIndex, pageSize)
	var list []model.SysApi
	if err := tx.Order(defaultOrder(q.Order, "api_id asc")).Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// GetApi 按主键查询接口。
func (r *Repo) GetApi(id int) (*model.SysApi, error) {
	var m model.SysApi
	if err := r.db.First(&m, "api_id = ?", id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// CreateApi 新增接口。
func (r *Repo) CreateApi(m *model.SysApi) error {
	if m.Type == "" {
		m.Type = "BUS"
	}
	return r.db.Create(m).Error
}

// UpdateApi 修改接口，并重建受影响角色的放行规则。
func (r *Repo) UpdateApi(m *model.SysApi) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.SysApi{}).Where("api_id = ?", m.ApiId).
			Select("path", "handle", "title", "type", "action").Updates(m).Error; err != nil {
			return err
		}
		return SyncAllRoleRules(tx)
	})
}

// DeleteApis 批量删除接口，并清理菜单关系与放行规则。
func (r *Repo) DeleteApis(ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("api_id IN ?", ids).Delete(&model.SysApi{}).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM sys_menu_api_rule WHERE api_id IN ?", ids).Error; err != nil {
			return err
		}
		return SyncAllRoleRules(tx)
	})
}

// Allow 判断角色是否可访问指定接口；角色标识为 admin 时全部放行。
func (r *Repo) Allow(roleKey, path, method string) (bool, error) {
	if roleKey == AdminRoleKey {
		return true, nil
	}
	var count int64
	err := r.db.Model(&model.CasbinRule{}).
		Where("ptype = ? AND v0 = ? AND v1 = ? AND v2 = ?", "p", roleKey, path, method).
		Count(&count).Error
	return count > 0, err
}

// CreateLoginLog 写入登录审计。
func (r *Repo) CreateLoginLog(m *model.SysLoginLog) error {
	return r.db.Create(m).Error
}

// CreateOperaLog 写入操作审计。
func (r *Repo) CreateOperaLog(m *model.SysOperaLog) error {
	return r.db.Create(m).Error
}

// RoleForUser 查询用户所属角色；用户未绑定角色时返回 nil。
func (r *Repo) RoleForUser(userId int) (*model.SysRole, error) {
	user, err := r.GetUser(userId)
	if err != nil {
		return nil, err
	}
	var role model.SysRole
	if err := r.db.First(&role, "role_id = ?", user.RoleId).Error; err != nil {
		if errRecordNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

// ConfigQuery 配置列表过滤条件。
type ConfigQuery struct {
	ConfigName string
	ConfigKey  string
	ConfigType string
	Order      string
}

// ListConfigs 分页查询系统配置。
func (r *Repo) ListConfigs(q ConfigQuery, pageIndex, pageSize int) ([]model.SysConfig, int64, error) {
	tx := r.db.Model(&model.SysConfig{})
	if q.ConfigName != "" {
		tx = tx.Where("config_name LIKE ?", like(q.ConfigName))
	}
	if q.ConfigKey != "" {
		tx = tx.Where("config_key LIKE ?", like(q.ConfigKey))
	}
	if q.ConfigType != "" {
		tx = tx.Where("config_type = ?", q.ConfigType)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset, limit := normalizePage(pageIndex, pageSize)
	var list []model.SysConfig
	if err := tx.Order(defaultOrder(q.Order, "config_id asc")).Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// GetConfig 按主键查询配置。
func (r *Repo) GetConfig(id int) (*model.SysConfig, error) {
	var m model.SysConfig
	if err := r.db.First(&m, "config_id = ?", id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// GetConfigByKey 按配置键查询配置。
func (r *Repo) GetConfigByKey(key string) (*model.SysConfig, error) {
	var m model.SysConfig
	if err := r.db.First(&m, "config_key = ?", key).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// CreateConfig 新增配置。
func (r *Repo) CreateConfig(m *model.SysConfig) error {
	if m.ConfigType == "" {
		m.ConfigType = "N"
	}
	return r.db.Create(m).Error
}

// UpdateConfig 修改配置。
func (r *Repo) UpdateConfig(m *model.SysConfig) error {
	return r.db.Model(&model.SysConfig{}).Where("config_id = ?", m.ConfigId).
		Select("config_name", "config_key", "config_value", "config_type", "is_frontend", "remark").
		Updates(m).Error
}

// DeleteConfigs 批量删除配置。
func (r *Repo) DeleteConfigs(ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Where("config_id IN ?", ids).Delete(&model.SysConfig{}).Error
}

// ConfigValue 按配置键取值，缺省返回 fallback。
func (r *Repo) ConfigValue(key, fallback string) string {
	var m model.SysConfig
	if err := r.db.First(&m, "config_key = ?", key).Error; err != nil {
		return fallback
	}
	if m.ConfigValue == "" {
		return fallback
	}
	return m.ConfigValue
}

// FrontendConfigs 前端可读的系统配置（登录页/布局展示用）。
func (r *Repo) FrontendConfigs() (map[string]string, error) {
	var list []model.SysConfig
	if err := r.db.Where("is_frontend = ?", "1").Find(&list).Error; err != nil {
		return nil, err
	}
	result := make(map[string]string, len(list))
	for _, m := range list {
		result[m.ConfigKey] = m.ConfigValue
	}
	return result, nil
}

// ConfigValues 按给定键批量取值；不存在的键返回空串。
func (r *Repo) ConfigValues(keys []string) (map[string]string, error) {
	var list []model.SysConfig
	if err := r.db.Where("config_key IN ?", keys).Find(&list).Error; err != nil {
		return nil, err
	}
	result := make(map[string]string, len(keys))
	for _, k := range keys {
		result[k] = ""
	}
	for _, m := range list {
		result[m.ConfigKey] = m.ConfigValue
	}
	return result, nil
}

// SetConfigs 批量写入配置：存在则更新，不存在则新增。
func (r *Repo) SetConfigs(entries map[string]string) error {
	for key, value := range entries {
		var m model.SysConfig
		err := r.db.First(&m, "config_key = ?", key).Error
		if err != nil && !errRecordNotFound(err) {
			return err
		}
		if errRecordNotFound(err) {
			m = model.SysConfig{ConfigKey: key, ConfigName: key, ConfigValue: value}
			if err := r.db.Create(&m).Error; err != nil {
				return err
			}
			continue
		}
		if err := r.db.Model(&model.SysConfig{}).Where("config_key = ?", key).
			Update("config_value", value).Error; err != nil {
			return err
		}
	}
	return nil
}

// DictTypeQuery 字典类型过滤条件。
type DictTypeQuery struct {
	DictName string
	DictType string
	Status   string
	Order    string
}

// DictDataQuery 字典项过滤条件。
type DictDataQuery struct {
	DictType  string
	DictLabel string
	Status    string
	Order     string
}

// ListDictTypes 分页查询字典类型。
func (r *Repo) ListDictTypes(q DictTypeQuery, pageIndex, pageSize int) ([]model.SysDictType, int64, error) {
	tx := r.db.Model(&model.SysDictType{})
	if q.DictName != "" {
		tx = tx.Where("dict_name LIKE ?", like(q.DictName))
	}
	if q.DictType != "" {
		tx = tx.Where("dict_type LIKE ?", like(q.DictType))
	}
	if q.Status != "" {
		tx = tx.Where("status = ?", q.Status)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset, limit := normalizePage(pageIndex, pageSize)
	var list []model.SysDictType
	if err := tx.Order(defaultOrder(q.Order, "dict_id asc")).Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// GetDictType 按主键查询字典类型。
func (r *Repo) GetDictType(id int) (*model.SysDictType, error) {
	var m model.SysDictType
	if err := r.db.First(&m, "dict_id = ?", id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// CreateDictType 新增字典类型。
func (r *Repo) CreateDictType(m *model.SysDictType) error {
	if m.Status == "" {
		m.Status = "2"
	}
	return r.db.Create(m).Error
}

// UpdateDictType 修改字典类型。
func (r *Repo) UpdateDictType(m *model.SysDictType) error {
	return r.db.Model(&model.SysDictType{}).Where("dict_id = ?", m.DictId).
		Select("dict_name", "dict_type", "status", "remark").Updates(m).Error
}

// DeleteDictTypes 批量删除字典类型及其字典项。
func (r *Repo) DeleteDictTypes(ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	var types []model.SysDictType
	if err := r.db.Where("dict_id IN ?", ids).Find(&types).Error; err != nil {
		return err
	}
	if err := r.db.Where("dict_id IN ?", ids).Delete(&model.SysDictType{}).Error; err != nil {
		return err
	}
	for _, t := range types {
		if err := r.db.Where("dict_type = ?", t.DictType).Delete(&model.SysDictData{}).Error; err != nil {
			return err
		}
	}
	return nil
}

// DictTypeOptions 字典类型下拉项。
func (r *Repo) DictTypeOptions() ([]model.SysDictType, error) {
	var list []model.SysDictType
	err := r.db.Order("dict_id asc").Find(&list).Error
	return list, err
}

// ListDictData 分页查询字典项。
func (r *Repo) ListDictData(q DictDataQuery, pageIndex, pageSize int) ([]model.SysDictData, int64, error) {
	tx := r.db.Model(&model.SysDictData{})
	if q.DictType != "" {
		tx = tx.Where("dict_type = ?", q.DictType)
	}
	if q.DictLabel != "" {
		tx = tx.Where("dict_label LIKE ?", like(q.DictLabel))
	}
	if q.Status != "" {
		tx = tx.Where("status = ?", q.Status)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset, limit := normalizePage(pageIndex, pageSize)
	var list []model.SysDictData
	if err := tx.Order(defaultOrder(q.Order, "dict_sort asc")).Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// GetDictData 按主键查询字典项。
func (r *Repo) GetDictData(code int) (*model.SysDictData, error) {
	var m model.SysDictData
	if err := r.db.First(&m, "dict_code = ?", code).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// CreateDictData 新增字典项。
func (r *Repo) CreateDictData(m *model.SysDictData) error {
	if m.Status == "" {
		m.Status = "2"
	}
	return r.db.Create(m).Error
}

// UpdateDictData 修改字典项。
func (r *Repo) UpdateDictData(m *model.SysDictData) error {
	return r.db.Model(&model.SysDictData{}).Where("dict_code = ?", m.DictCode).
		Select("dict_sort", "dict_label", "dict_value", "dict_type", "css_class", "list_class", "is_default", "status", "remark").
		Updates(m).Error
}

// DeleteDictData 批量删除字典项。
func (r *Repo) DeleteDictData(ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Where("dict_code IN ?", ids).Delete(&model.SysDictData{}).Error
}

// DictDataOptions 指定字典类型的下拉项（只含启用项）。
func (r *Repo) DictDataOptions(dictType string) ([]model.SysDictData, error) {
	var list []model.SysDictData
	err := r.db.Where("dict_type = ? AND status = ?", dictType, "2").Order("dict_sort asc").Find(&list).Error
	return list, err
}

// LoginLogQuery 登录日志过滤条件。
type LoginLogQuery struct {
	Username  string
	Ipaddr    string
	Status    string
	BeginTime string
	EndTime   string
	Order     string
}

// OperaLogQuery 操作日志过滤条件。
type OperaLogQuery struct {
	Title     string
	OperName  string
	Status    string
	BeginTime string
	EndTime   string
	Order     string
}

// ListLoginLogs 分页查询登录日志。
func (r *Repo) ListLoginLogs(q LoginLogQuery, pageIndex, pageSize int) ([]model.SysLoginLog, int64, error) {
	tx := r.db.Model(&model.SysLoginLog{})
	if q.Username != "" {
		tx = tx.Where("username LIKE ?", like(q.Username))
	}
	if q.Ipaddr != "" {
		tx = tx.Where("ipaddr LIKE ?", like(q.Ipaddr))
	}
	if q.Status != "" {
		tx = tx.Where("status = ?", q.Status)
	}
	if q.BeginTime != "" {
		tx = tx.Where("login_time >= ?", q.BeginTime)
	}
	if q.EndTime != "" {
		tx = tx.Where("login_time <= ?", q.EndTime)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset, limit := normalizePage(pageIndex, pageSize)
	var list []model.SysLoginLog
	if err := tx.Order(defaultOrder(q.Order, "info_id desc")).Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// GetLoginLog 按主键查询登录日志。
func (r *Repo) GetLoginLog(id int) (*model.SysLoginLog, error) {
	var m model.SysLoginLog
	if err := r.db.First(&m, "info_id = ?", id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// DeleteLoginLogs 批量删除登录日志。
func (r *Repo) DeleteLoginLogs(ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Where("info_id IN ?", ids).Delete(&model.SysLoginLog{}).Error
}

// ListOperaLogs 分页查询操作日志。
func (r *Repo) ListOperaLogs(q OperaLogQuery, pageIndex, pageSize int) ([]model.SysOperaLog, int64, error) {
	tx := r.db.Model(&model.SysOperaLog{})
	if q.Title != "" {
		tx = tx.Where("title LIKE ?", like(q.Title))
	}
	if q.OperName != "" {
		tx = tx.Where("oper_name LIKE ?", like(q.OperName))
	}
	if q.Status != "" {
		tx = tx.Where("status = ?", q.Status)
	}
	if q.BeginTime != "" {
		tx = tx.Where("oper_time >= ?", q.BeginTime)
	}
	if q.EndTime != "" {
		tx = tx.Where("oper_time <= ?", q.EndTime)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset, limit := normalizePage(pageIndex, pageSize)
	var list []model.SysOperaLog
	if err := tx.Order(defaultOrder(q.Order, "oper_id desc")).Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// DeleteOperaLogs 批量删除操作日志。
func (r *Repo) DeleteOperaLogs(ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Where("oper_id IN ?", ids).Delete(&model.SysOperaLog{}).Error
}

// CleanOperaLogs 清空操作日志。
func (r *Repo) CleanOperaLogs() error {
	return r.db.Where("oper_id > ?", 0).Delete(&model.SysOperaLog{}).Error
}

// MenuQuery 菜单列表过滤条件。
type MenuQuery struct {
	Title    string
	MenuName string
	Visible  string
}

// MenuNode 前端菜单树节点（角色授权树使用）。
type MenuNode struct {
	Id       int        `json:"id"`
	Label    string     `json:"label"`
	Children []MenuNode `json:"children,omitempty"`
}

// ListMenus 查询菜单并组装为树。
func (r *Repo) ListMenus(q MenuQuery) ([]model.SysMenu, error) {
	tx := r.db.Model(&model.SysMenu{})
	if q.Title != "" {
		tx = tx.Where("title LIKE ?", like(q.Title))
	}
	if q.MenuName != "" {
		tx = tx.Where("menu_name LIKE ?", like(q.MenuName))
	}
	if q.Visible != "" {
		tx = tx.Where("visible = ?", q.Visible)
	}
	var list []model.SysMenu
	if err := tx.Order("sort asc").Find(&list).Error; err != nil {
		return nil, err
	}
	if err := r.fillMenuApis(list); err != nil {
		return nil, err
	}
	return buildMenuTree(list, 0), nil
}

// GetMenu 查询单个菜单，带出已勾选的接口 id。
func (r *Repo) GetMenu(menuId int) (*model.SysMenu, error) {
	var m model.SysMenu
	if err := r.db.Preload("SysApi").First(&m, "menu_id = ?", menuId).Error; err != nil {
		return nil, err
	}
	m.Apis = apiIds(m.SysApi)
	return &m, nil
}

// CreateMenu 新增菜单并写入菜单-接口关系。
func (r *Repo) CreateMenu(m *model.SysMenu, apis []int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(m).Error; err != nil {
			return err
		}
		if err := replaceMenuApis(tx, m.MenuId, apis); err != nil {
			return err
		}
		return SyncAllRoleRules(tx)
	})
}

// UpdateMenu 修改菜单并重写菜单-接口关系。
func (r *Repo) UpdateMenu(m *model.SysMenu, apis []int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.SysMenu{}).Where("menu_id = ?", m.MenuId).
			Select("menu_name", "title", "icon", "path", "menu_type", "action", "permission",
				"parent_id", "no_cache", "component", "sort", "visible", "is_frame").
			Updates(m).Error; err != nil {
			return err
		}
		if err := replaceMenuApis(tx, m.MenuId, apis); err != nil {
			return err
		}
		return SyncAllRoleRules(tx)
	})
}

// DeleteMenus 批量删除菜单（含子孙），并清理角色关系与放行规则。
func (r *Repo) DeleteMenus(ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		all := collectMenuIds(tx, ids)
		if err := tx.Where("menu_id IN ?", all).Delete(&model.SysMenu{}).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM sys_role_menu WHERE menu_id IN ?", all).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM sys_menu_api_rule WHERE menu_id IN ?", all).Error; err != nil {
			return err
		}
		return SyncAllRoleRules(tx)
	})
}

// RoleMenuTreeselect 角色授权树：roleId 为 0 时只返回全量菜单树（用于新增角色）。
func (r *Repo) RoleMenuTreeselect(roleId int) ([]MenuNode, []int, error) {
	var menus []model.SysMenu
	if err := r.db.Order("sort asc").Find(&menus).Error; err != nil {
		return nil, nil, err
	}
	checked := make([]int, 0)
	if roleId > 0 {
		owned, err := r.roleMenus(roleId)
		if err != nil {
			return nil, nil, err
		}
		checked = menuIds(owned)
	}
	return buildMenuNodeTree(menus, 0), checked, nil
}

// AdminRoleKey 超级管理员角色标识：持有全部权限，无需菜单授权。
const AdminRoleKey = "admin"

// MenusForRoute 当前角色的路由菜单树；按钮型（F）不参与路由。
func (r *Repo) MenusForRoute(role *model.SysRole) ([]model.SysMenu, error) {
	var (
		menus []model.SysMenu
		err   error
	)
	if role.RoleKey == AdminRoleKey {
		err = r.db.Order("sort asc").Find(&menus).Error
	} else {
		menus, err = r.roleMenus(role.RoleId)
	}
	if err != nil {
		return nil, err
	}
	visible := make([]model.SysMenu, 0, len(menus))
	for _, m := range menus {
		if m.MenuType == "F" {
			continue
		}
		if m.MenuType == "M" && m.Component == "" {
			m.Component = "Layout"
		}
		visible = append(visible, m)
	}
	return buildMenuTree(visible, 0), nil
}

// PermissionCodes 角色菜单上的权限标识集合。
func (r *Repo) PermissionCodes(role *model.SysRole) ([]string, error) {
	var codes []string
	if role.RoleKey == AdminRoleKey {
		err := r.db.Model(&model.SysMenu{}).Where("permission <> ''").Pluck("permission", &codes).Error
		return codes, err
	}
	err := r.db.Table("sys_menu").
		Joins("JOIN sys_role_menu ON sys_role_menu.menu_id = sys_menu.menu_id").
		Where("sys_role_menu.role_id = ? AND sys_menu.permission <> ''", role.RoleId).
		Pluck("sys_menu.permission", &codes).Error
	return codes, err
}

// fillMenuApis 为菜单列表填充 sysApi。
func (r *Repo) fillMenuApis(menus []model.SysMenu) error {
	if len(menus) == 0 {
		return nil
	}
	ids := make([]int, 0, len(menus))
	for _, m := range menus {
		ids = append(ids, m.MenuId)
	}
	var apis []model.SysApi
	if err := r.db.Table("sys_api").
		Joins("JOIN sys_menu_api_rule ON sys_menu_api_rule.api_id = sys_api.api_id").
		Where("sys_menu_api_rule.menu_id IN ?", ids).
		Order("sys_api.api_id asc").
		Find(&apis).Error; err != nil {
		return err
	}
	owned := make(map[int][]model.SysApi)
	var rules []struct {
		MenuId int
		ApiId  int
	}
	if err := r.db.Table("sys_menu_api_rule").Where("menu_id IN ?", ids).Scan(&rules).Error; err != nil {
		return err
	}
	byId := make(map[int]model.SysApi, len(apis))
	for _, a := range apis {
		byId[a.ApiId] = a
	}
	for _, rule := range rules {
		if api, ok := byId[rule.ApiId]; ok {
			owned[rule.MenuId] = append(owned[rule.MenuId], api)
		}
	}
	for i := range menus {
		menus[i].SysApi = owned[menus[i].MenuId]
		menus[i].Apis = apiIds(menus[i].SysApi)
	}
	return nil
}

// replaceMenuApis 覆盖菜单勾选的接口。
func replaceMenuApis(tx *gorm.DB, menuId int, apis []int) error {
	if err := tx.Exec("DELETE FROM sys_menu_api_rule WHERE menu_id = ?", menuId).Error; err != nil {
		return err
	}
	if len(apis) == 0 {
		return nil
	}
	rows := make([]map[string]any, 0, len(apis))
	seen := make(map[int]struct{}, len(apis))
	for _, apiId := range apis {
		if _, ok := seen[apiId]; ok {
			continue
		}
		seen[apiId] = struct{}{}
		rows = append(rows, map[string]any{"menu_id": menuId, "api_id": apiId})
	}
	return tx.Table("sys_menu_api_rule").Create(rows).Error
}

// collectMenuIds 收集 ids 及其所有子孙菜单 id。
func collectMenuIds(tx *gorm.DB, ids []int) []int {
	all := append([]int{}, ids...)
	seen := make(map[int]struct{}, len(ids))
	for _, id := range ids {
		seen[id] = struct{}{}
	}
	frontier := ids
	for len(frontier) > 0 {
		var children []int
		if err := tx.Model(&model.SysMenu{}).Where("parent_id IN ?", frontier).Pluck("menu_id", &children).Error; err != nil {
			break
		}
		frontier = nil
		for _, id := range children {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			all = append(all, id)
			frontier = append(frontier, id)
		}
	}
	return all
}

// buildMenuTree 由扁平菜单构建指定父节点的子树。
func buildMenuTree(list []model.SysMenu, parentId int) []model.SysMenu {
	tree := make([]model.SysMenu, 0)
	for _, m := range list {
		if m.ParentId != parentId {
			continue
		}
		m.Children = buildMenuTree(list, m.MenuId)
		tree = append(tree, m)
	}
	return tree
}

// buildMenuNodeTree 构建角色授权用的 id/label 树。
func buildMenuNodeTree(list []model.SysMenu, parentId int) []MenuNode {
	sort.SliceStable(list, func(i, j int) bool { return list[i].Sort < list[j].Sort })
	nodes := make([]MenuNode, 0)
	for _, m := range list {
		if m.ParentId != parentId {
			continue
		}
		nodes = append(nodes, MenuNode{
			Id:       m.MenuId,
			Label:    m.Title,
			Children: buildMenuNodeTree(list, m.MenuId),
		})
	}
	return nodes
}

// apiIds 提取接口主键。
func apiIds(apis []model.SysApi) []int {
	ids := make([]int, 0, len(apis))
	for _, a := range apis {
		ids = append(ids, a.ApiId)
	}
	return ids
}

// RoleQuery 角色列表过滤条件。
type RoleQuery struct {
	RoleName string
	RoleKey  string
	Status   string
	Order    string
}

// ListRoles 分页查询角色。
func (r *Repo) ListRoles(q RoleQuery, pageIndex, pageSize int) ([]model.SysRole, int64, error) {
	tx := r.db.Model(&model.SysRole{})
	if q.RoleName != "" {
		tx = tx.Where("role_name LIKE ?", like(q.RoleName))
	}
	if q.RoleKey != "" {
		tx = tx.Where("role_key LIKE ?", like(q.RoleKey))
	}
	if q.Status != "" {
		tx = tx.Where("status = ?", q.Status)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset, limit := normalizePage(pageIndex, pageSize)
	var list []model.SysRole
	if err := tx.Order(defaultOrder(q.Order, "role_sort asc")).Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// GetRole 按主键查询角色，并带出已分配的菜单 id。
func (r *Repo) GetRole(roleId int) (*model.SysRole, error) {
	var m model.SysRole
	if err := r.db.First(&m, "role_id = ?", roleId).Error; err != nil {
		return nil, err
	}
	menus, err := r.roleMenus(roleId)
	if err != nil {
		return nil, err
	}
	m.MenuIds = menuIds(menus)
	return &m, nil
}

// GetRoleByKey 按 roleKey 查询角色。
func (r *Repo) GetRoleByKey(roleKey string) (*model.SysRole, error) {
	var m model.SysRole
	if err := r.db.First(&m, "role_key = ?", roleKey).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// CreateRole 新增角色并同步菜单与放行规则。
func (r *Repo) CreateRole(m *model.SysRole, menuIds []int) error {
	if m.Status == "" {
		m.Status = "2"
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(m).Error; err != nil {
			return err
		}
		if err := replaceRoleMenus(tx, m.RoleId, menuIds); err != nil {
			return err
		}
		return syncRoleRules(tx, m.RoleId, m.RoleKey)
	})
}

// UpdateRole 修改角色；菜单变更后重建该角色的放行规则。
func (r *Repo) UpdateRole(m *model.SysRole, menuIds []int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var old model.SysRole
		if err := tx.First(&old, "role_id = ?", m.RoleId).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.SysRole{}).Where("role_id = ?", m.RoleId).
			Select("role_name", "role_key", "role_sort", "status", "remark").Updates(m).Error; err != nil {
			return err
		}
		if err := replaceRoleMenus(tx, m.RoleId, menuIds); err != nil {
			return err
		}
		// roleKey 变更时先清掉旧键的规则
		if old.RoleKey != m.RoleKey {
			if err := tx.Where("ptype = ? AND v0 = ?", "p", old.RoleKey).Delete(&model.CasbinRule{}).Error; err != nil {
				return err
			}
		}
		return syncRoleRules(tx, m.RoleId, m.RoleKey)
	})
}

// DeleteRoles 批量删除角色并清理其菜单关系与放行规则。
func (r *Repo) DeleteRoles(ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		var roles []model.SysRole
		if err := tx.Where("role_id IN ?", ids).Find(&roles).Error; err != nil {
			return err
		}
		if err := tx.Where("role_id IN ?", ids).Delete(&model.SysRole{}).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM sys_role_menu WHERE role_id IN ?", ids).Error; err != nil {
			return err
		}
		for _, role := range roles {
			if err := tx.Where("ptype = ? AND v0 = ?", "p", role.RoleKey).Delete(&model.CasbinRule{}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// UpdateRoleStatus 启用/停用角色。
func (r *Repo) UpdateRoleStatus(roleId int, status string) error {
	return r.db.Model(&model.SysRole{}).Where("role_id = ?", roleId).Update("status", status).Error
}

// roleMenus 查询角色已分配的菜单。
func (r *Repo) roleMenus(roleId int) ([]model.SysMenu, error) {
	var menus []model.SysMenu
	err := r.db.Table("sys_menu").
		Joins("JOIN sys_role_menu ON sys_role_menu.menu_id = sys_menu.menu_id").
		Where("sys_role_menu.role_id = ?", roleId).
		Order("sys_menu.sort asc").
		Find(&menus).Error
	return menus, err
}

// replaceRoleMenus 覆盖角色的菜单关系。
func replaceRoleMenus(tx *gorm.DB, roleId int, ids []int) error {
	if err := tx.Exec("DELETE FROM sys_role_menu WHERE role_id = ?", roleId).Error; err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	var menus []model.SysMenu
	if err := tx.Where("menu_id IN ?", ids).Find(&menus).Error; err != nil {
		return err
	}
	if len(menus) == 0 {
		return nil
	}
	rows := make([]map[string]any, 0, len(menus))
	for _, m := range menus {
		rows = append(rows, map[string]any{"role_id": roleId, "menu_id": m.MenuId})
	}
	return tx.Table("sys_role_menu").Create(rows).Error
}

// syncRoleRules 按角色的菜单-接口关系重建 casbin_rule。
func syncRoleRules(tx *gorm.DB, roleId int, roleKey string) error {
	if err := tx.Where("ptype = ? AND v0 = ?", "p", roleKey).Delete(&model.CasbinRule{}).Error; err != nil {
		return err
	}
	var apis []model.SysApi
	if err := tx.Table("sys_api").
		Joins("JOIN sys_menu_api_rule ON sys_menu_api_rule.api_id = sys_api.api_id").
		Joins("JOIN sys_role_menu ON sys_role_menu.menu_id = sys_menu_api_rule.menu_id").
		Where("sys_role_menu.role_id = ?", roleId).
		Group("sys_api.api_id, sys_api.path, sys_api.action").
		Find(&apis).Error; err != nil {
		return err
	}
	rules := make([]model.CasbinRule, 0, len(apis))
	for _, a := range apis {
		if a.Path == "" || a.Action == "" {
			continue
		}
		rules = append(rules, model.CasbinRule{Ptype: "p", V0: roleKey, V1: a.Path, V2: a.Action})
	}
	if len(rules) == 0 {
		return nil
	}
	return tx.CreateInBatches(rules, 100).Error
}

// SyncAllRoleRules 全量重建放行规则；菜单/接口变更后调用，避免漏掉受影响角色。
func SyncAllRoleRules(tx *gorm.DB) error {
	var roles []model.SysRole
	if err := tx.Find(&roles).Error; err != nil {
		return err
	}
	for _, role := range roles {
		if role.RoleKey == AdminRoleKey {
			continue // 超级管理员直接放行，不写规则
		}
		if err := syncRoleRules(tx, role.RoleId, role.RoleKey); err != nil {
			return err
		}
	}
	return nil
}

// menuIds 提取菜单主键。
func menuIds(menus []model.SysMenu) []int {
	ids := make([]int, 0, len(menus))
	for _, m := range menus {
		ids = append(ids, m.MenuId)
	}
	return ids
}

// UserQuery 用户列表过滤条件。
type UserQuery struct {
	Username string
	Phone    string
	Status   string
	Order    string
}

// ListUsers 分页查询用户。
func (r *Repo) ListUsers(q UserQuery, pageIndex, pageSize int) ([]model.SysUser, int64, error) {
	tx := r.db.Model(&model.SysUser{})
	if q.Username != "" {
		tx = tx.Where("username LIKE ?", like(q.Username))
	}
	if q.Phone != "" {
		tx = tx.Where("phone LIKE ?", like(q.Phone))
	}
	if q.Status != "" {
		tx = tx.Where("status = ?", q.Status)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset, limit := normalizePage(pageIndex, pageSize)
	var list []model.SysUser
	if err := tx.Order(defaultOrder(q.Order, "user_id desc")).Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// GetUser 按主键查询用户。
func (r *Repo) GetUser(userId int) (*model.SysUser, error) {
	var m model.SysUser
	if err := r.db.First(&m, "user_id = ?", userId).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// FindUserByUsername 按用户名查询用户。
func (r *Repo) FindUserByUsername(username string) (*model.SysUser, error) {
	var m model.SysUser
	if err := r.db.First(&m, "username = ?", username).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// CreateUser 新增用户；密码为空时使用默认初始密码。
func (r *Repo) CreateUser(m *model.SysUser) error {
	if m.Status == "" {
		m.Status = "2"
	}
	if m.Password == "" {
		m.Password = r.ConfigValue("sys_user_initPassword", "123456")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(m.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	m.Password = string(hash)
	return r.db.Create(m).Error
}

// UpdateUser 修改用户；密码留空表示不修改。
func (r *Repo) UpdateUser(m *model.SysUser) error {
	tx := r.db.Model(m)
	if m.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(m.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		m.Password = string(hash)
		return tx.Select("username", "nick_name", "phone", "role_id", "avatar", "sex", "email", "remark", "status", "password").Updates(m).Error
	}
	return tx.Select("username", "nick_name", "phone", "role_id", "avatar", "sex", "email", "remark", "status").Updates(m).Error
}

// DeleteUsers 批量删除用户。
func (r *Repo) DeleteUsers(ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Where("user_id IN ?", ids).Delete(&model.SysUser{}).Error
}

// UpdateUserStatus 启用/停用用户。
func (r *Repo) UpdateUserStatus(userId int, status string) error {
	return r.db.Model(&model.SysUser{}).Where("user_id = ?", userId).Update("status", status).Error
}

// ResetUserPassword 重置指定用户密码。
func (r *Repo) ResetUserPassword(userId int, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return r.db.Model(&model.SysUser{}).Where("user_id = ?", userId).Update("password", string(hash)).Error
}

// CheckUserPassword 校验用户明文密码。
func CheckUserPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// UpdateUserProfile 修改当前登录用户的资料（昵称/手机/邮箱/性别）。
func (r *Repo) UpdateUserProfile(m *model.SysUser) error {
	return r.db.Model(&model.SysUser{}).Where("user_id = ?", m.UserId).
		Select("nick_name", "phone", "email", "sex").Updates(m).Error
}

// errRecordNotFound 判断是否记录不存在。
func errRecordNotFound(err error) bool { return err == gorm.ErrRecordNotFound }
