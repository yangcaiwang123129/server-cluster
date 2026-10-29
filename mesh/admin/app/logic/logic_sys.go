package logic

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dobyte/jwt"

	"server-cluster/internal/svc"
	"server-cluster/mesh/admin/app/repo"
	"server-cluster/share/consts"
	"server-cluster/share/dto"
	"github.com/dobyte/due/v2/codes"
	"server-cluster/share/model"
)

// ApiList 接口分页列表。
func ApiList(req dto.ApiListReq) ([]model.SysApi, int64, error) {
	return repo.R().ListApis(repo.ApiQuery{
		Title:  req.Title,
		Path:   req.Path,
		Action: req.Action,
		Type:   req.Type,
		Order:  req.Order,
	}, req.PageIndex, req.PageSize)
}

// ApiGet 接口详情。
func ApiGet(id int) (*model.SysApi, error) { return repo.R().GetApi(id) }

// ApiCreate 新增接口。
func ApiCreate(req dto.ApiSaveReq) (*model.SysApi, error) {
	if req.Path == "" || req.Action == "" {
		return nil, codes.NewCode(400, "接口路径与方法不能为空").Err()
	}
	m := apiModel(req)
	if err := repo.R().CreateApi(m); err != nil {
		return nil, err
	}
	return m, nil
}

// ApiUpdate 修改接口。
func ApiUpdate(id int, req dto.ApiSaveReq) (*model.SysApi, error) {
	if id == 0 {
		return nil, codes.NewCode(400, "接口ID不能为空").Err()
	}
	m := apiModel(req)
	m.ApiId = id
	if err := repo.R().UpdateApi(m); err != nil {
		return nil, err
	}
	return m, nil
}

// ApiDelete 批量删除接口。
func ApiDelete(ids []int) error { return repo.R().DeleteApis(ids) }

// apiModel 请求体转模型。
func apiModel(req dto.ApiSaveReq) *model.SysApi {
	return &model.SysApi{
		Path:   req.Path,
		Handle: req.Handle,
		Title:  req.Title,
		Type:   req.Type,
		Action: req.Action,
	}
}

// ---------------- 图形验证码（进程内，3 分钟有效） ----------------

type captchaItem struct {
	code   string
	expire time.Time
}

var (
	captchaMu    sync.Mutex
	captchaItems = make(map[string]captchaItem)
)

const (
	captchaTTL  = 3 * time.Minute
	captchaSize = 4
)

// Captcha 生成验证码，返回 id 与 data URI。
func Captcha() (id, uri string) {
	code := randomDigits(captchaSize)
	id = randomHex(16)
	captchaMu.Lock()
	defer captchaMu.Unlock()
	now := time.Now()
	for k, item := range captchaItems {
		if now.After(item.expire) {
			delete(captchaItems, k)
		}
	}
	captchaItems[id] = captchaItem{code: code, expire: now.Add(captchaTTL)}
	return id, "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(captchaSVG(code)))
}

// verifyCaptcha 校验并立即作废验证码。
func verifyCaptcha(id, code string) bool {
	if id == "" || code == "" {
		return false
	}
	captchaMu.Lock()
	defer captchaMu.Unlock()
	item, ok := captchaItems[id]
	delete(captchaItems, id)
	if !ok || time.Now().After(item.expire) {
		return false
	}
	return strings.EqualFold(item.code, code)
}

// captchaSVG 绘制验证码图片。
func captchaSVG(code string) string {
	var b strings.Builder
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" width="120" height="40" viewBox="0 0 120 40">`)
	b.WriteString(`<rect width="120" height="40" fill="#f2f3f5"/>`)
	colors := [4]string{"#4a4a4a", "#2f6fa8", "#8a5a2b", "#2f7a4a"}
	for i := range code {
		x := 14 + i*27
		y := 29
		rotate := randomInt(28) - 14
		fmt.Fprintf(&b, `<text x="%d" y="%d" font-size="26" font-family="monospace" font-weight="bold" fill="%s" transform="rotate(%d %d %d)">%c</text>`,
			x, y, colors[i%len(colors)], rotate, x, y, code[i])
	}
	for i := 0; i < 3; i++ {
		fmt.Fprintf(&b, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#9aa0a6" stroke-width="1"/>`,
			randomInt(40), randomInt(40), 80+randomInt(40), randomInt(40))
	}
	b.WriteString(`</svg>`)
	return b.String()
}

// randomDigits 生成指定长度的数字串。
func randomDigits(n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteByte(byte('0' + randomInt(10)))
	}
	return b.String()
}

// randomHex 生成十六进制随机串（验证码 id）。
func randomHex(n int) string {
	buf := make([]byte, n/2)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return fmt.Sprintf("%x", buf)
}

// randomInt 返回 [0,n) 的随机数。
func randomInt(n int) int {
	if n <= 0 {
		return 0
	}
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0
	}
	return int(v.Int64())
}

// ---------------- 登录与鉴权 ----------------

// Login 校验验证码与账号密码，签发令牌。
func Login(req dto.AdminLoginReq, ip, ua string) (string, error) {
	if !verifyCaptcha(req.UUID, req.Code) {
		return "", codes.NewCode(400, "验证码错误或已过期").Err()
	}
	user, err := repo.R().FindUserByUsername(req.Username)
	if err != nil || !repo.CheckUserPassword(user.Password, req.Password) {
		RecordLogin(req.Username, ip, ua, false, "用户名或密码错误")
		return "", codes.NewCode(400, "用户名或密码错误").Err()
	}
	if user.Status != consts.StatusEnabled {
		RecordLogin(req.Username, ip, ua, false, "账号已停用")
		return "", codes.NewCode(400, "账号已停用").Err()
	}
	role, err := repo.R().RoleForUser(user.UserId)
	if err != nil || role == nil {
		RecordLogin(req.Username, ip, ua, false, "账号未绑定角色")
		return "", codes.NewCode(400, "账号未绑定角色").Err()
	}
	if role.Status != consts.StatusEnabled {
		RecordLogin(req.Username, ip, ua, false, "角色已停用")
		return "", codes.NewCode(400, "角色已停用").Err()
	}
	token, err := svc.JWT().GenerateToken(jwt.Payload{"uid": user.UserId})
	if err != nil {
		return "", codes.NewCode(500, "令牌签发失败").Err()
	}
	RecordLogin(req.Username, ip, ua, true, "登录成功")
	return token.Token, nil
}

// GetInfo 当前登录用户的角色、权限与资料。
func GetInfo(user *model.SysUser, role *model.SysRole) dto.GetInfoResp {
	permissions := make([]string, 0)
	roles := make([]string, 0)
	if role != nil {
		roles = append(roles, role.RoleKey)
		if codes, err := PermissionCodes(role); err == nil {
			permissions = codes
		}
	}
	name := user.NickName
	if name == "" {
		name = user.Username
	}
	return dto.GetInfoResp{
		Roles:        roles,
		Name:         name,
		Avatar:       user.Avatar,
		Introduction: user.Remark,
		Permissions:  permissions,
	}
}

// MenuRole 当前用户的动态路由菜单树。
func MenuRole(role *model.SysRole) ([]model.SysMenu, error) {
	if role == nil {
		return nil, codes.NewCode(403, "账号未绑定角色").Err()
	}
	return RouteMenus(role)
}

// settingKeys 系统配置页展示的内置键。
var settingKeys = []string{
	consts.ConfigKeyAppName,
	consts.ConfigKeyAppLogo,
	consts.ConfigKeyInitPassword,
	consts.ConfigKeyIndexSkinName,
	consts.ConfigKeyIndexSideTheme,
}

// ConfigList 系统配置分页列表。
func ConfigList(req dto.ConfigListReq) ([]model.SysConfig, int64, error) {
	return repo.R().ListConfigs(repo.ConfigQuery{
		ConfigName: req.ConfigName,
		ConfigKey:  req.ConfigKey,
		ConfigType: req.ConfigType,
		Order:      req.Order,
	}, req.PageIndex, req.PageSize)
}

// ConfigGet 配置详情。
func ConfigGet(id int) (*model.SysConfig, error) { return repo.R().GetConfig(id) }

// ConfigGetByKey 按配置键查配置。
func ConfigGetByKey(key string) (*model.SysConfig, error) { return repo.R().GetConfigByKey(key) }

// ConfigCreate 新增配置。
func ConfigCreate(req dto.ConfigSaveReq) (*model.SysConfig, error) {
	if req.ConfigKey == "" {
		return nil, codes.NewCode(400, "配置键不能为空").Err()
	}
	m := configModel(req)
	if err := repo.R().CreateConfig(m); err != nil {
		return nil, err
	}
	return m, nil
}

// ConfigUpdate 修改配置。
func ConfigUpdate(id int, req dto.ConfigSaveReq) (*model.SysConfig, error) {
	if id == 0 {
		return nil, codes.NewCode(400, "配置ID不能为空").Err()
	}
	m := configModel(req)
	m.ConfigId = id
	if err := repo.R().UpdateConfig(m); err != nil {
		return nil, err
	}
	return m, nil
}

// ConfigDelete 批量删除配置。
func ConfigDelete(ids []int) error { return repo.R().DeleteConfigs(ids) }

// FrontendConfigs 前端可读配置（登录页/布局展示用）。
func FrontendConfigs() (map[string]string, error) { return repo.R().FrontendConfigs() }

// SettingValues 系统配置页的内置键取值。
func SettingValues() (map[string]string, error) { return repo.R().ConfigValues(settingKeys) }

// UpdateSettings 批量保存系统配置页的内置键。
func UpdateSettings(req dto.SetConfigReq) error {
	values := make(map[string]string, len(req))
	for _, item := range req {
		if item.ConfigKey != "" {
			values[item.ConfigKey] = item.ConfigValue
		}
	}
	return repo.R().SetConfigs(values)
}

// configModel 请求体转模型。
func configModel(req dto.ConfigSaveReq) *model.SysConfig {
	return &model.SysConfig{
		ConfigName:  req.ConfigName,
		ConfigKey:   req.ConfigKey,
		ConfigValue: req.ConfigValue,
		ConfigType:  req.ConfigType,
		IsFrontend:  req.IsFrontend,
		Remark:      req.Remark,
	}
}

// DictTypeList 字典类型分页列表。
func DictTypeList(req dto.DictTypeListReq) ([]model.SysDictType, int64, error) {
	return repo.R().ListDictTypes(repo.DictTypeQuery{
		DictName: req.DictName,
		DictType: req.DictType,
		Status:   req.Status,
		Order:    req.Order,
	}, req.PageIndex, req.PageSize)
}

// DictTypeGet 字典类型详情。
func DictTypeGet(id int) (*model.SysDictType, error) { return repo.R().GetDictType(id) }

// DictTypeCreate 新增字典类型。
func DictTypeCreate(req dto.DictTypeSaveReq) (*model.SysDictType, error) {
	if req.DictName == "" || req.DictType == "" {
		return nil, codes.NewCode(400, "字典名称与类型不能为空").Err()
	}
	m := &model.SysDictType{
		DictName: req.DictName,
		DictType: req.DictType,
		Status:   statusText(req.Status),
		Remark:   req.Remark,
	}
	if err := repo.R().CreateDictType(m); err != nil {
		return nil, err
	}
	return m, nil
}

// DictTypeUpdate 修改字典类型。
func DictTypeUpdate(id int, req dto.DictTypeSaveReq) (*model.SysDictType, error) {
	if id == 0 {
		return nil, codes.NewCode(400, "字典ID不能为空").Err()
	}
	m := &model.SysDictType{
		DictId:   id,
		DictName: req.DictName,
		DictType: req.DictType,
		Status:   statusText(req.Status),
		Remark:   req.Remark,
	}
	if err := repo.R().UpdateDictType(m); err != nil {
		return nil, err
	}
	return m, nil
}

// DictTypeDelete 批量删除字典类型（级联删除字典项）。
func DictTypeDelete(ids []int) error { return repo.R().DeleteDictTypes(ids) }

// DictTypeOptions 字典类型下拉项。
func DictTypeOptions() ([]dto.Option, error) {
	list, err := repo.R().DictTypeOptions()
	if err != nil {
		return nil, err
	}
	options := make([]dto.Option, 0, len(list))
	for _, item := range list {
		options = append(options, dto.Option{Label: item.DictName, Value: item.DictType})
	}
	return options, nil
}

// DictDataList 字典项分页列表。
func DictDataList(req dto.DictDataListReq) ([]model.SysDictData, int64, error) {
	return repo.R().ListDictData(repo.DictDataQuery{
		DictType:  req.DictType,
		DictLabel: req.DictLabel,
		Status:    req.Status,
		Order:     req.Order,
	}, req.PageIndex, req.PageSize)
}

// DictDataGet 字典项详情。
func DictDataGet(code int) (*model.SysDictData, error) { return repo.R().GetDictData(code) }

// DictDataCreate 新增字典项。
func DictDataCreate(req dto.DictDataSaveReq) (*model.SysDictData, error) {
	if req.DictType == "" || req.DictLabel == "" {
		return nil, codes.NewCode(400, "字典类型与标签不能为空").Err()
	}
	m := dictDataModel(req, 0)
	if err := repo.R().CreateDictData(m); err != nil {
		return nil, err
	}
	return m, nil
}

// DictDataUpdate 修改字典项。
func DictDataUpdate(code int, req dto.DictDataSaveReq) (*model.SysDictData, error) {
	if code == 0 {
		return nil, codes.NewCode(400, "字典项ID不能为空").Err()
	}
	m := dictDataModel(req, code)
	if err := repo.R().UpdateDictData(m); err != nil {
		return nil, err
	}
	return m, nil
}

// DictDataDelete 批量删除字典项。
func DictDataDelete(ids []int) error { return repo.R().DeleteDictData(ids) }

// DictDataOptions 指定字典类型的下拉项。
func DictDataOptions(dictType string) ([]dto.Option, error) {
	list, err := repo.R().DictDataOptions(dictType)
	if err != nil {
		return nil, err
	}
	options := make([]dto.Option, 0, len(list))
	for _, item := range list {
		options = append(options, dto.Option{Label: item.DictLabel, Value: item.DictValue})
	}
	return options, nil
}

// statusText 数字状态转库内字符串（1 停用 / 2 正常，0 视为正常）。
func statusText(status int) string {
	if status <= 0 {
		return consts.StatusEnabled
	}
	return strconv.Itoa(status)
}

// dictDataModel 请求体转模型。
func dictDataModel(req dto.DictDataSaveReq, dictCode int) *model.SysDictData {
	isDefault := req.IsDefault
	if isDefault == "" {
		isDefault = consts.FlagNo
	}
	return &model.SysDictData{
		DictCode:  dictCode,
		DictSort:  req.DictSort,
		DictLabel: req.DictLabel,
		DictValue: req.DictValue,
		DictType:  req.DictType,
		CssClass:  req.CssClass,
		ListClass: req.ListClass,
		IsDefault: isDefault,
		Status:    statusText(req.Status),
		Remark:    req.Remark,
	}
}

// RecordLogin 写入登录审计；失败不影响主流程。
func RecordLogin(username, ip, ua string, success bool, msg string) {
	browser, osName := parseUserAgent(ua)
	status := consts.LogStatusFail
	if success {
		status = consts.LogStatusSuccess
	}
	_ = repo.R().CreateLoginLog(&model.SysLoginLog{
		Username:      username,
		Ipaddr:        ip,
		LoginLocation: "内网",
		Browser:       browser,
		Os:            osName,
		Platform:      "Web",
		Status:        status,
		Msg:           msg,
		LoginTime:     time.Now(),
	})
}

// LoginLogList 登录日志分页列表。
func LoginLogList(req dto.LoginLogListReq) ([]model.SysLoginLog, int64, error) {
	return repo.R().ListLoginLogs(repo.LoginLogQuery{
		Username:  req.Username,
		Ipaddr:    req.Ipaddr,
		Status:    req.Status,
		BeginTime: req.BeginTime,
		EndTime:   req.EndTime,
		Order:     req.Order,
	}, req.PageIndex, req.PageSize)
}

// LoginLogGet 登录日志详情。
func LoginLogGet(id int) (*model.SysLoginLog, error) { return repo.R().GetLoginLog(id) }

// LoginLogDelete 批量删除登录日志。
func LoginLogDelete(ids []int) error { return repo.R().DeleteLoginLogs(ids) }

// OperaLogList 操作日志分页列表。
func OperaLogList(req dto.OperaLogListReq) ([]model.SysOperaLog, int64, error) {
	return repo.R().ListOperaLogs(repo.OperaLogQuery{
		Title:     req.Title,
		OperName:  req.OperName,
		Status:    req.Status,
		BeginTime: req.BeginTime,
		EndTime:   req.EndTime,
		Order:     req.Order,
	}, req.PageIndex, req.PageSize)
}

// OperaLogDelete 批量删除操作日志。
func OperaLogDelete(ids []int) error { return repo.R().DeleteOperaLogs(ids) }

// OperaLogClean 清空操作日志。
func OperaLogClean() error { return repo.R().CleanOperaLogs() }

// parseUserAgent 粗略识别浏览器与操作系统。
func parseUserAgent(ua string) (browser, osName string) {
	browser, osName = "Unknown", "Unknown"
	switch {
	case strings.Contains(ua, "Edg"):
		browser = "Edge"
	case strings.Contains(ua, "Chrome"):
		browser = "Chrome"
	case strings.Contains(ua, "Firefox"):
		browser = "Firefox"
	case strings.Contains(ua, "Safari"):
		browser = "Safari"
	}
	switch {
	case strings.Contains(ua, "Windows"):
		osName = "Windows"
	case strings.Contains(ua, "Mac OS"):
		osName = "MacOS"
	case strings.Contains(ua, "Android"):
		osName = "Android"
	case strings.Contains(ua, "Linux"):
		osName = "Linux"
	}
	return
}

// MenuList 菜单树。
func MenuList(req dto.MenuListReq) ([]model.SysMenu, error) {
	return repo.R().ListMenus(repo.MenuQuery{
		Title:    req.Title,
		MenuName: req.MenuName,
		Visible:  req.Visible,
	})
}

// MenuGet 菜单详情（含已勾选接口）。
func MenuGet(id int) (*model.SysMenu, error) { return repo.R().GetMenu(id) }

// MenuCreate 新增菜单与接口授权关系。
func MenuCreate(req dto.MenuSaveReq) (*model.SysMenu, error) {
	m := menuModel(req)
	if err := repo.R().CreateMenu(m, req.Apis); err != nil {
		return nil, err
	}
	return m, nil
}

// MenuUpdate 修改菜单与接口授权关系。
func MenuUpdate(id int, req dto.MenuSaveReq) (*model.SysMenu, error) {
	if id == 0 {
		return nil, codes.NewCode(400, "菜单ID不能为空").Err()
	}
	m := menuModel(req)
	m.MenuId = id
	if err := repo.R().UpdateMenu(m, req.Apis); err != nil {
		return nil, err
	}
	return m, nil
}

// MenuDelete 批量删除菜单（含子孙）。
func MenuDelete(ids []int) error { return repo.R().DeleteMenus(ids) }

// RoleMenuTreeselect 角色授权树。
func RoleMenuTreeselect(roleId int) (dto.RoleMenuTreeselectResp, error) {
	menus, checked, err := repo.R().RoleMenuTreeselect(roleId)
	if err != nil {
		return dto.RoleMenuTreeselectResp{}, err
	}
	nodes := make([]dto.MenuNode, 0, len(menus))
	for _, n := range menus {
		nodes = append(nodes, menuNode(n))
	}
	return dto.RoleMenuTreeselectResp{Menus: nodes, CheckedKeys: checked}, nil
}

// RouteMenus 当前角色的路由菜单树。
func RouteMenus(role *model.SysRole) ([]model.SysMenu, error) { return repo.R().MenusForRoute(role) }

// PermissionCodes 角色菜单上的权限标识集合。
func PermissionCodes(role *model.SysRole) ([]string, error) { return repo.R().PermissionCodes(role) }

// menuModel 请求体转模型。
func menuModel(req dto.MenuSaveReq) *model.SysMenu {
	return &model.SysMenu{
		MenuName:   req.MenuName,
		Title:      req.Title,
		Icon:       req.Icon,
		Path:       req.Path,
		MenuType:   req.MenuType,
		Action:     req.Action,
		Permission: req.Permission,
		ParentId:   req.ParentId,
		NoCache:    req.NoCache,
		Component:  req.Component,
		Sort:       req.Sort,
		Visible:    req.Visible,
		IsFrame:    req.IsFrame,
		Apis:       req.Apis,
	}
}

// menuNode 仓库树节点转 DTO 节点。
func menuNode(n repo.MenuNode) dto.MenuNode {
	node := dto.MenuNode{Id: n.Id, Label: n.Label}
	if len(n.Children) > 0 {
		node.Children = make([]dto.MenuNode, 0, len(n.Children))
		for _, child := range n.Children {
			node.Children = append(node.Children, menuNode(child))
		}
	}
	return node
}

// RoleList 角色分页列表。
func RoleList(req dto.RoleListReq) ([]model.SysRole, int64, error) {
	return repo.R().ListRoles(repo.RoleQuery{
		RoleName: req.RoleName,
		RoleKey:  req.RoleKey,
		Status:   req.Status,
		Order:    req.Order,
	}, req.PageIndex, req.PageSize)
}

// RoleGet 角色详情（含已分配菜单）。
func RoleGet(id int) (*model.SysRole, error) { return repo.R().GetRole(id) }

// RoleCreate 新增角色。
func RoleCreate(req dto.RoleSaveReq) (*model.SysRole, error) {
	if req.RoleName == "" || req.RoleKey == "" {
		return nil, codes.NewCode(400, "角色名称与角色标识不能为空").Err()
	}
	if req.RoleKey == consts.AdminRoleKey {
		return nil, codes.NewCode(400, "角色标识 admin 为系统保留").Err()
	}
	m := roleModel(req)
	if err := repo.R().CreateRole(m, req.MenuIds); err != nil {
		return nil, err
	}
	return m, nil
}

// RoleUpdate 修改角色与菜单授权。
func RoleUpdate(id int, req dto.RoleSaveReq) (*model.SysRole, error) {
	if id == 0 {
		return nil, codes.NewCode(400, "角色ID不能为空").Err()
	}
	m := roleModel(req)
	m.RoleId = id
	if err := repo.R().UpdateRole(m, req.MenuIds); err != nil {
		return nil, err
	}
	return m, nil
}

// RoleDelete 批量删除角色；超级管理员角色不可删除。
func RoleDelete(ids []int) error {
	adminRole, err := repo.R().GetRoleByKey(consts.AdminRoleKey)
	if err == nil {
		for _, id := range ids {
			if id == adminRole.RoleId {
				return codes.NewCode(400, "超级管理员角色不可删除").Err()
			}
		}
	}
	return repo.R().DeleteRoles(ids)
}

// RoleUpdateStatus 启用/停用角色。
func RoleUpdateStatus(req dto.RoleStatusReq) error {
	return repo.R().UpdateRoleStatus(req.RoleId, req.Status)
}

// roleModel 请求体转模型。
func roleModel(req dto.RoleSaveReq) *model.SysRole {
	return &model.SysRole{
		RoleName: req.RoleName,
		RoleKey:  req.RoleKey,
		RoleSort: req.RoleSort,
		Status:   req.Status,
		Remark:   req.Remark,
		MenuIds:  req.MenuIds,
	}
}

// UserList 用户分页列表。
func UserList(req dto.UserListReq) ([]model.SysUser, int64, error) {
	return repo.R().ListUsers(repo.UserQuery{
		Username: req.Username,
		Phone:    req.Phone,
		Status:   req.Status,
		Order:    req.Order,
	}, req.PageIndex, req.PageSize)
}

// UserGet 用户详情。
func UserGet(id int) (*model.SysUser, error) { return repo.R().GetUser(id) }

// UserCreate 新增用户。
func UserCreate(req dto.UserSaveReq) (*model.SysUser, error) {
	if req.Username == "" {
		return nil, codes.NewCode(400, "用户名不能为空").Err()
	}
	m := userModel(req)
	if err := repo.R().CreateUser(m); err != nil {
		return nil, err
	}
	return m, nil
}

// UserUpdate 修改用户。
func UserUpdate(id int, req dto.UserSaveReq) (*model.SysUser, error) {
	if id == 0 {
		return nil, codes.NewCode(400, "用户ID不能为空").Err()
	}
	m := userModel(req)
	m.UserId = id
	if err := repo.R().UpdateUser(m); err != nil {
		return nil, err
	}
	return m, nil
}

// UserDelete 批量删除用户。
func UserDelete(ids []int) error { return repo.R().DeleteUsers(ids) }

// UserUpdateStatus 启用/停用用户。
func UserUpdateStatus(req dto.UserStatusReq) error {
	return repo.R().UpdateUserStatus(req.UserId, req.Status)
}

// UserResetPassword 管理员重置指定用户密码。
func UserResetPassword(req dto.UserPasswordResetReq) error {
	if req.Password == "" {
		return codes.NewCode(400, "密码不能为空").Err()
	}
	return repo.R().ResetUserPassword(req.UserId, req.Password)
}

// UserUpdateProfile 修改当前登录用户的资料。
func UserUpdateProfile(userId int, req dto.UserProfileUpdateReq) (*model.SysUser, error) {
	m := &model.SysUser{
		UserId:   userId,
		NickName: req.NickName,
		Phone:    req.Phone,
		Email:    req.Email,
		Sex:      req.Sex,
	}
	if err := repo.R().UpdateUserProfile(m); err != nil {
		return nil, err
	}
	return m, nil
}

// UserSetPassword 当前登录用户修改密码。
func UserSetPassword(user *model.SysUser, req dto.UserPasswordSetReq) error {
	if req.NewPassword == "" {
		return codes.NewCode(400, "新密码不能为空").Err()
	}
	if !repo.CheckUserPassword(user.Password, req.OldPassword) {
		return codes.NewCode(400, "原密码错误").Err()
	}
	return repo.R().ResetUserPassword(user.UserId, req.NewPassword)
}

// userModel 请求体转模型。
func userModel(req dto.UserSaveReq) *model.SysUser {
	return &model.SysUser{
		Username: req.Username,
		Password: req.Password,
		NickName: req.NickName,
		Phone:    req.Phone,
		RoleId:   req.RoleId,
		Avatar:   req.Avatar,
		Sex:      req.Sex,
		Email:    req.Email,
		Remark:   req.Remark,
		Status:   req.Status,
	}
}
