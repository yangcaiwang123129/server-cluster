package handler

import (
	"github.com/dobyte/due/component/http/v2"
	"github.com/gofiber/fiber/v3"

	"server-cluster/mesh/admin/app/globol"
	"server-cluster/mesh/admin/app/logic"
	"server-cluster/share/dto"
	"server-cluster/share/errs"
)

// registerApi 接口登记模块路由。
func registerApi(secure http.Router) {
	secure.Get("/sys-api", handleApiList)
	secure.Get("/sys-api/:id", handleApiGet)
	secure.Post("/sys-api", handleApiCreate)
	secure.Put("/sys-api/:id", handleApiUpdate)
	secure.Delete("/sys-api", handleApiDelete)
}

// handleApiList 接口分页列表。
func handleApiList(ctx http.Context) error {
	var req dto.ApiListReq
	req.PageIndex, req.PageSize = pageParams(ctx)
	req.Title = ctx.Query("title")
	req.Path = ctx.Query("path")
	req.Action = ctx.Query("action")
	req.Type = ctx.Query("type")
	req.Order = orderBy(ctx, map[string]string{
		"title":     "title",
		"createdAt": "created_at",
	})
	list, total, err := logic.ApiList(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, errs.Page(list, total, req.PageIndex, req.PageSize))
}

// handleApiGet 接口详情。
func handleApiGet(ctx http.Context) error {
	m, err := logic.ApiGet(paramInt(ctx, "id"))
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, m)
}

// handleApiCreate 新增接口。
func handleApiCreate(ctx http.Context) error {
	var req dto.ApiSaveReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	m, err := logic.ApiCreate(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, m)
}

// handleApiUpdate 修改接口。
func handleApiUpdate(ctx http.Context) error {
	var req dto.ApiSaveReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	m, err := logic.ApiUpdate(paramInt(ctx, "id"), req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, m)
}

// handleApiDelete 批量删除接口。
func handleApiDelete(ctx http.Context) error {
	var req dto.IdsReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	if err := logic.ApiDelete(req.Ids); err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OKMsg(ctx, "删除成功")
}

// registerAuth 认证模块路由。
func registerAuth(router http.Router) {
	router.Get("/captcha", handleCaptcha)
	router.Post("/login", handleLogin)
	router.Get("/app-config", handleAppConfig)

	router.Post("/logout", handleLogout)
	router.Get("/getinfo", handleGetInfo)
	router.Get("/menurole", handleMenuRole)
}

// handleCaptcha 返回图形验证码。
func handleCaptcha(ctx http.Context) error {
	id, uri := logic.Captcha()
	return ctx.Status(fiber.StatusOK).JSON(dto.CaptchaResp{Code: 200, Msg: "ok", Id: id, Data: uri})
}

// handleLogin 账号密码 + 图形验证码登录。
func handleLogin(ctx http.Context) error {
	var req dto.AdminLoginReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	token, err := logic.Login(req, ctx.IP(), ctx.Get("User-Agent"))
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return ctx.Status(fiber.StatusOK).JSON(dto.TokenResp{Code: 200, Msg: "登录成功", Token: token})
}

// handleLogout 注销（JWT 无状态，仅返回成功）。
func handleLogout(ctx http.Context) error { return errs.OKMsg(ctx, "注销成功") }

// handleGetInfo 返回当前登录用户的角色、权限与资料。
func handleGetInfo(ctx http.Context) error {
	return errs.OK(ctx, logic.GetInfo(globol.CurrentUser(ctx), globol.CurrentRole(ctx)))
}

// handleMenuRole 当前用户的动态路由菜单树。
func handleMenuRole(ctx http.Context) error {
	menus, err := logic.MenuRole(globol.CurrentRole(ctx))
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, menus)
}

// handleAppConfig 登录页/布局使用的品牌配置（无需鉴权）。
func handleAppConfig(ctx http.Context) error {
	configs, err := logic.FrontendConfigs()
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, configs)
}

// registerConfig 系统配置模块路由。
func registerConfig(secure http.Router) {
	secure.Get("/config", handleConfigList)
	secure.Get("/config/:id", handleConfigGet)
	secure.Get("/configKey/:key", handleConfigKey)
	secure.Post("/config", handleConfigCreate)
	secure.Put("/config/:id", handleConfigUpdate)
	secure.Delete("/config", handleConfigDelete)
	secure.Get("/set-config", handleGetSetConfig)
	secure.Put("/set-config", handleUpdateSetConfig)
}

// handleConfigList 系统配置分页列表。
func handleConfigList(ctx http.Context) error {
	var req dto.ConfigListReq
	req.PageIndex, req.PageSize = pageParams(ctx)
	req.ConfigName = ctx.Query("configName")
	req.ConfigKey = ctx.Query("configKey")
	req.ConfigType = ctx.Query("configType")
	req.Order = orderBy(ctx, map[string]string{
		"createdAt": "created_at",
	})
	list, total, err := logic.ConfigList(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, errs.Page(list, total, req.PageIndex, req.PageSize))
}

// handleConfigGet 配置详情。
func handleConfigGet(ctx http.Context) error {
	m, err := logic.ConfigGet(paramInt(ctx, "id"))
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, m)
}

// handleConfigKey 按配置键取值。
func handleConfigKey(ctx http.Context) error {
	m, err := logic.ConfigGetByKey(ctx.Params("key"))
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, dto.ConfigValueResp{ConfigValue: m.ConfigValue})
}

// handleConfigCreate 新增配置。
func handleConfigCreate(ctx http.Context) error {
	var req dto.ConfigSaveReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	m, err := logic.ConfigCreate(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, m)
}

// handleConfigUpdate 修改配置。
func handleConfigUpdate(ctx http.Context) error {
	var req dto.ConfigSaveReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	m, err := logic.ConfigUpdate(paramInt(ctx, "id"), req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, m)
}

// handleConfigDelete 批量删除配置。
func handleConfigDelete(ctx http.Context) error {
	var req dto.IdsReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	if err := logic.ConfigDelete(req.Ids); err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OKMsg(ctx, "删除成功")
}

// handleGetSetConfig 系统配置页的内置键取值。
func handleGetSetConfig(ctx http.Context) error {
	values, err := logic.SettingValues()
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, values)
}

// handleUpdateSetConfig 批量保存系统配置页的内置键。
func handleUpdateSetConfig(ctx http.Context) error {
	var req dto.SetConfigReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	if err := logic.UpdateSettings(req); err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OKMsg(ctx, "保存成功")
}

// registerDict 字典模块路由。
func registerDict(secure http.Router) {
	secure.Get("/dict/type", handleDictTypeList)
	secure.Get("/dict/type/:id", handleDictTypeGet)
	secure.Post("/dict/type", handleDictTypeCreate)
	secure.Put("/dict/type/:id", handleDictTypeUpdate)
	secure.Delete("/dict/type", handleDictTypeDelete)
	secure.Get("/dict/type-option-select", handleDictTypeOptions)
	secure.Get("/dict/data", handleDictDataList)
	secure.Get("/dict/data/:dictCode", handleDictDataGet)
	secure.Post("/dict/data", handleDictDataCreate)
	secure.Put("/dict/data/:dictCode", handleDictDataUpdate)
	secure.Delete("/dict/data", handleDictDataDelete)
	secure.Get("/dict-data/option-select", handleDictDataOptions)
}

// handleDictTypeList 字典类型分页列表。
func handleDictTypeList(ctx http.Context) error {
	var req dto.DictTypeListReq
	req.PageIndex, req.PageSize = pageParams(ctx)
	req.DictName = ctx.Query("dictName")
	req.DictType = ctx.Query("dictType")
	req.Status = ctx.Query("status")
	req.Order = orderBy(ctx, map[string]string{
		"createdAt": "created_at",
	})
	list, total, err := logic.DictTypeList(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, errs.Page(list, total, req.PageIndex, req.PageSize))
}

// handleDictTypeGet 字典类型详情。
func handleDictTypeGet(ctx http.Context) error {
	m, err := logic.DictTypeGet(paramInt(ctx, "id"))
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, m)
}

// handleDictTypeCreate 新增字典类型。
func handleDictTypeCreate(ctx http.Context) error {
	var req dto.DictTypeSaveReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	m, err := logic.DictTypeCreate(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, m)
}

// handleDictTypeUpdate 修改字典类型。
func handleDictTypeUpdate(ctx http.Context) error {
	var req dto.DictTypeSaveReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	m, err := logic.DictTypeUpdate(paramInt(ctx, "id"), req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, m)
}

// handleDictTypeDelete 批量删除字典类型。
func handleDictTypeDelete(ctx http.Context) error {
	var req dto.IdsReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	if err := logic.DictTypeDelete(req.Ids); err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OKMsg(ctx, "删除成功")
}

// handleDictTypeOptions 字典类型下拉项。
func handleDictTypeOptions(ctx http.Context) error {
	options, err := logic.DictTypeOptions()
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, options)
}

// handleDictDataList 字典项分页列表。
func handleDictDataList(ctx http.Context) error {
	var req dto.DictDataListReq
	req.PageIndex, req.PageSize = pageParams(ctx)
	req.DictType = ctx.Query("dictType")
	req.DictLabel = ctx.Query("dictLabel")
	req.Status = ctx.Query("status")
	req.Order = orderBy(ctx, map[string]string{
		"dictSort":  "dict_sort",
		"createdAt": "created_at",
	})
	list, total, err := logic.DictDataList(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, errs.Page(list, total, req.PageIndex, req.PageSize))
}

// handleDictDataGet 字典项详情。
func handleDictDataGet(ctx http.Context) error {
	m, err := logic.DictDataGet(paramInt(ctx, "dictCode"))
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, m)
}

// handleDictDataCreate 新增字典项。
func handleDictDataCreate(ctx http.Context) error {
	var req dto.DictDataSaveReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	m, err := logic.DictDataCreate(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, m)
}

// handleDictDataUpdate 修改字典项。
func handleDictDataUpdate(ctx http.Context) error {
	var req dto.DictDataSaveReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	m, err := logic.DictDataUpdate(paramInt(ctx, "dictCode"), req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, m)
}

// handleDictDataDelete 批量删除字典项。
func handleDictDataDelete(ctx http.Context) error {
	var req dto.IdsReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	if err := logic.DictDataDelete(req.Ids); err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OKMsg(ctx, "删除成功")
}

// handleDictDataOptions 指定字典类型的下拉项。
func handleDictDataOptions(ctx http.Context) error {
	options, err := logic.DictDataOptions(ctx.Query("dictType"))
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, options)
}

// registerLog 审计日志模块路由。
func registerLog(secure http.Router) {
	secure.Get("/sys-login-log", handleLoginLogList)
	secure.Get("/sys-login-log/:id", handleLoginLogGet)
	secure.Delete("/sys-login-log", handleLoginLogDelete)
	secure.Get("/sys-opera-log", handleOperaLogList)
	secure.Delete("/sys-opera-log", handleOperaLogDelete)
	secure.Delete("/operlog/clean", handleOperaLogClean)
}

// handleLoginLogList 登录日志分页列表。
func handleLoginLogList(ctx http.Context) error {
	var req dto.LoginLogListReq
	req.PageIndex, req.PageSize = pageParams(ctx)
	req.Username = ctx.Query("username")
	req.Ipaddr = ctx.Query("ipaddr")
	req.Status = ctx.Query("status")
	req.BeginTime = ctx.Query("beginTime")
	req.EndTime = ctx.Query("endTime")
	req.Order = orderBy(ctx, map[string]string{
		"loginTime": "login_time",
	})
	list, total, err := logic.LoginLogList(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, errs.Page(list, total, req.PageIndex, req.PageSize))
}

// handleLoginLogGet 登录日志详情。
func handleLoginLogGet(ctx http.Context) error {
	m, err := logic.LoginLogGet(paramInt(ctx, "id"))
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, m)
}

// handleLoginLogDelete 批量删除登录日志。
func handleLoginLogDelete(ctx http.Context) error {
	var req dto.IdsReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	if err := logic.LoginLogDelete(req.Ids); err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OKMsg(ctx, "删除成功")
}

// handleOperaLogList 操作日志分页列表。
func handleOperaLogList(ctx http.Context) error {
	var req dto.OperaLogListReq
	req.PageIndex, req.PageSize = pageParams(ctx)
	req.Title = ctx.Query("title")
	req.OperName = ctx.Query("operName")
	req.Status = ctx.Query("status")
	req.BeginTime = ctx.Query("beginTime")
	req.EndTime = ctx.Query("endTime")
	req.Order = orderBy(ctx, map[string]string{
		"operTime": "oper_time",
	})
	list, total, err := logic.OperaLogList(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, errs.Page(list, total, req.PageIndex, req.PageSize))
}

// handleOperaLogDelete 批量删除操作日志。
func handleOperaLogDelete(ctx http.Context) error {
	var req dto.IdsReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	if err := logic.OperaLogDelete(req.Ids); err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OKMsg(ctx, "删除成功")
}

// handleOperaLogClean 清空操作日志。
func handleOperaLogClean(ctx http.Context) error {
	if err := logic.OperaLogClean(); err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OKMsg(ctx, "清空成功")
}

// registerMenu 菜单模块路由。
func registerMenu(secure http.Router) {
	secure.Get("/menu", handleMenuList)
	secure.Get("/menu/:menuId", handleMenuGet)
	secure.Get("/roleMenuTreeselect/:roleId", handleRoleMenuTreeselect)
	secure.Post("/menu", handleMenuCreate)
	secure.Put("/menu/:id", handleMenuUpdate)
	secure.Delete("/menu", handleMenuDelete)
}

// handleMenuList 菜单树（不分页）。
func handleMenuList(ctx http.Context) error {
	menus, err := logic.MenuList(dto.MenuListReq{
		Title:    ctx.Query("title"),
		MenuName: ctx.Query("menuName"),
		Visible:  ctx.Query("visible"),
	})
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, menus)
}

// handleMenuGet 菜单详情（含已勾选接口）。
func handleMenuGet(ctx http.Context) error {
	m, err := logic.MenuGet(paramInt(ctx, "menuId"))
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, m)
}

// handleRoleMenuTreeselect 角色授权树。
func handleRoleMenuTreeselect(ctx http.Context) error {
	resp, err := logic.RoleMenuTreeselect(paramInt(ctx, "roleId"))
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, resp)
}

// handleMenuCreate 新增菜单。
func handleMenuCreate(ctx http.Context) error {
	var req dto.MenuSaveReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	m, err := logic.MenuCreate(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, m)
}

// handleMenuUpdate 修改菜单。
func handleMenuUpdate(ctx http.Context) error {
	var req dto.MenuSaveReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	m, err := logic.MenuUpdate(paramInt(ctx, "id"), req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, m)
}

// handleMenuDelete 批量删除菜单（含子孙）。
func handleMenuDelete(ctx http.Context) error {
	var req dto.IdsReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	if err := logic.MenuDelete(req.Ids); err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OKMsg(ctx, "删除成功")
}

// registerRole 角色模块路由。
func registerRole(secure http.Router) {
	secure.Get("/role", handleRoleList)
	secure.Get("/role/:roleId", handleRoleGet)
	secure.Post("/role", handleRoleCreate)
	secure.Put("/role/:roleId", handleRoleUpdate)
	secure.Delete("/role", handleRoleDelete)
	secure.Put("/role-status", handleRoleStatus)
}

// handleRoleList 角色分页列表。
func handleRoleList(ctx http.Context) error {
	var req dto.RoleListReq
	req.PageIndex, req.PageSize = pageParams(ctx)
	req.RoleName = ctx.Query("roleName")
	req.RoleKey = ctx.Query("roleKey")
	req.Status = ctx.Query("status")
	req.Order = orderBy(ctx, map[string]string{
		"roleSort":  "role_sort",
		"createdAt": "created_at",
	})
	list, total, err := logic.RoleList(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, errs.Page(list, total, req.PageIndex, req.PageSize))
}

// handleRoleGet 角色详情（含已分配菜单）。
func handleRoleGet(ctx http.Context) error {
	m, err := logic.RoleGet(paramInt(ctx, "roleId"))
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, m)
}

// handleRoleCreate 新增角色。
func handleRoleCreate(ctx http.Context) error {
	var req dto.RoleSaveReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	m, err := logic.RoleCreate(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, m)
}

// handleRoleUpdate 修改角色与菜单授权。
func handleRoleUpdate(ctx http.Context) error {
	var req dto.RoleSaveReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	m, err := logic.RoleUpdate(paramInt(ctx, "roleId"), req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, m)
}

// handleRoleDelete 批量删除角色。
func handleRoleDelete(ctx http.Context) error {
	var req dto.IdsReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	if err := logic.RoleDelete(req.Ids); err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OKMsg(ctx, "删除成功")
}

// handleRoleStatus 启用/停用角色。
func handleRoleStatus(ctx http.Context) error {
	var req dto.RoleStatusReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	if err := logic.RoleUpdateStatus(req); err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OKMsg(ctx, "操作成功")
}

// registerUser 用户模块路由。
func registerUser(secure http.Router) {
	secure.Get("/sys-user", handleUserList)
	secure.Get("/sys-user/:userId", handleUserGet)
	secure.Post("/sys-user", handleUserCreate)
	secure.Put("/sys-user", handleUserUpdate)
	secure.Delete("/sys-user", handleUserDelete)
	secure.Put("/user/status", handleUserStatus)
	secure.Put("/user/pwd/reset", handleUserResetPwd)
	secure.Get("/user/profile", handleUserProfile)
	secure.Put("/sys-user/profile", handleUserProfileUpdate)
	secure.Put("/user/pwd/set", handleUserPwdSet)
}

// handleUserList 用户分页列表。
func handleUserList(ctx http.Context) error {
	var req dto.UserListReq
	req.PageIndex, req.PageSize = pageParams(ctx)
	req.Username = ctx.Query("username")
	req.Phone = ctx.Query("phone")
	req.Status = ctx.Query("status")
	req.Order = orderBy(ctx, map[string]string{
		"username":  "username",
		"createdAt": "created_at",
		"status":    "status",
	})
	list, total, err := logic.UserList(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, errs.Page(list, total, req.PageIndex, req.PageSize))
}

// handleUserGet 用户详情。
func handleUserGet(ctx http.Context) error {
	m, err := logic.UserGet(paramInt(ctx, "userId"))
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, m)
}

// handleUserCreate 新增用户。
func handleUserCreate(ctx http.Context) error {
	var req dto.UserSaveReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	m, err := logic.UserCreate(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, m)
}

// handleUserUpdate 修改用户。
func handleUserUpdate(ctx http.Context) error {
	var req dto.UserSaveReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	m, err := logic.UserUpdate(req.UserId, req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, m)
}

// handleUserDelete 批量删除用户。
func handleUserDelete(ctx http.Context) error {
	var req dto.IdsReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	if err := logic.UserDelete(req.Ids); err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OKMsg(ctx, "删除成功")
}

// handleUserStatus 启用/停用用户。
func handleUserStatus(ctx http.Context) error {
	var req dto.UserStatusReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	if err := logic.UserUpdateStatus(req); err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OKMsg(ctx, "操作成功")
}

// handleUserResetPwd 重置指定用户密码。
func handleUserResetPwd(ctx http.Context) error {
	var req dto.UserPasswordResetReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	if err := logic.UserResetPassword(req); err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OKMsg(ctx, "密码重置成功")
}

// handleUserProfile 当前登录用户的资料。
func handleUserProfile(ctx http.Context) error {
	roleName := ""
	if role := globol.CurrentRole(ctx); role != nil {
		roleName = role.RoleName
	}
	return errs.OK(ctx, dto.UserProfileResp{User: globol.CurrentUser(ctx), RoleName: roleName})
}

// handleUserProfileUpdate 修改当前登录用户的资料。
func handleUserProfileUpdate(ctx http.Context) error {
	var req dto.UserProfileUpdateReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	m, err := logic.UserUpdateProfile(globol.CurrentUser(ctx).UserId, req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, m)
}

// handleUserPwdSet 当前登录用户修改密码。
func handleUserPwdSet(ctx http.Context) error {
	var req dto.UserPasswordSetReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	if err := logic.UserSetPassword(globol.CurrentUser(ctx), req); err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OKMsg(ctx, "密码修改成功")
}
