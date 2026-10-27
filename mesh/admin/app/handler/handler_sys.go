package handler

import (
	"github.com/dobyte/due/component/http/v2"
	"github.com/gofiber/fiber/v3"

	"server-cluster/mesh/admin/app/global"
	"server-cluster/mesh/admin/app/logic"
	"server-cluster/share/dto"
	"server-cluster/share/errs"
)

// RegisterSys 注册管理后台权限管理路由。
func RegisterSys(router http.Router) {
	groupRouter := router.Group("/api/v1",
		global.CORSMiddleware(),
		global.RequireDB,
		global.OperaLogMiddleware,
		global.AuthMiddleware,
		global.PermissionMiddleware,
	)

	groupRouter.Get("/sys-api", handleApiList)
	groupRouter.Get("/sys-api/:id", handleApiGet)
	groupRouter.Post("/sys-api", handleApiCreate)
	groupRouter.Put("/sys-api/:id", handleApiUpdate)
	groupRouter.Delete("/sys-api", handleApiDelete)

	// registerAuth 认证模块路由。
	groupRouter.Get("/captcha", handleCaptcha)
	groupRouter.Post("/login", handleLogin)
	groupRouter.Get("/app-config", handleAppConfig)

	groupRouter.Post("/logout", handleLogout)
	groupRouter.Get("/getinfo", handleGetInfo)
	groupRouter.Get("/menurole", handleMenuRole)

	// registerUser 用户模块路由。
	groupRouter.Get("/sys-user", handleUserList)
	groupRouter.Get("/sys-user/:userId", handleUserGet)
	groupRouter.Post("/sys-user", handleUserCreate)
	groupRouter.Put("/sys-user", handleUserUpdate)
	groupRouter.Delete("/sys-user", handleUserDelete)
	groupRouter.Put("/user/status", handleUserStatus)
	groupRouter.Put("/user/pwd/reset", handleUserResetPwd)
	groupRouter.Get("/user/profile", handleUserProfile)
	groupRouter.Put("/sys-user/profile", handleUserProfileUpdate)
	groupRouter.Put("/user/pwd/set", handleUserPwdSet)

	// registerRole 角色模块路由。
	groupRouter.Get("/role", handleRoleList)
	groupRouter.Get("/role/:roleId", handleRoleGet)
	groupRouter.Post("/role", handleRoleCreate)
	groupRouter.Put("/role/:roleId", handleRoleUpdate)
	groupRouter.Delete("/role", handleRoleDelete)
	groupRouter.Put("/role-status", handleRoleStatus)

	// registerMenu 菜单模块路由。
	groupRouter.Get("/menu", handleMenuList)
	groupRouter.Get("/menu/:menuId", handleMenuGet)
	groupRouter.Get("/roleMenuTreeselect/:roleId", handleRoleMenuTreeselect)
	groupRouter.Post("/menu", handleMenuCreate)
	groupRouter.Put("/menu/:id", handleMenuUpdate)
	groupRouter.Delete("/menu", handleMenuDelete)

	// registerDict 字典模块路由。
	groupRouter.Get("/dict/type", handleDictTypeList)
	groupRouter.Get("/dict/type/:id", handleDictTypeGet)
	groupRouter.Post("/dict/type", handleDictTypeCreate)
	groupRouter.Put("/dict/type/:id", handleDictTypeUpdate)
	groupRouter.Delete("/dict/type", handleDictTypeDelete)
	groupRouter.Get("/dict/type-option-select", handleDictTypeOptions)
	groupRouter.Get("/dict/data", handleDictDataList)
	groupRouter.Get("/dict/data/:dictCode", handleDictDataGet)
	groupRouter.Post("/dict/data", handleDictDataCreate)
	groupRouter.Put("/dict/data/:dictCode", handleDictDataUpdate)
	groupRouter.Delete("/dict/data", handleDictDataDelete)
	groupRouter.Get("/dict-data/option-select", handleDictDataOptions)

	// registerConfig 系统配置模块路由。
	groupRouter.Get("/config", handleConfigList)
	groupRouter.Get("/config/:id", handleConfigGet)
	groupRouter.Get("/configKey/:key", handleConfigKey)
	groupRouter.Post("/config", handleConfigCreate)
	groupRouter.Put("/config/:id", handleConfigUpdate)
	groupRouter.Delete("/config", handleConfigDelete)
	groupRouter.Get("/set-config", handleGetSetConfig)
	groupRouter.Put("/set-config", handleUpdateSetConfig)

	// registerLog 审计日志模块路由。
	groupRouter.Get("/sys-login-log", handleLoginLogList)
	groupRouter.Get("/sys-login-log/:id", handleLoginLogGet)
	groupRouter.Delete("/sys-login-log", handleLoginLogDelete)
	groupRouter.Get("/sys-opera-log", handleOperaLogList)
	groupRouter.Delete("/sys-opera-log", handleOperaLogDelete)
	groupRouter.Delete("/operlog/clean", handleOperaLogClean)

	// registerPlayer 玩家管理路由。
	groupRouter.Get("/player", handlePlayerList)
	groupRouter.Get("/player/online", handlePlayerOnline)
}

// handleApiList 接口分页列表。
// @Summary 接口列表
// @Tags system
// @Produce json
// @Param pageIndex query int false "页码" default(1)
// @Param pageSize query int false "每页条数" default(10)
// @Param title query string false "接口标题"
// @Param path query string false "接口路径"
// @Param action query string false "请求方式"
// @Param type query string false "接口类型"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/sys-api [get]
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
// @Summary 接口详情
// @Tags system
// @Produce json
// @Param id path int true "接口ID"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/sys-api/{id} [get]
func handleApiGet(ctx http.Context) error {
	m, err := logic.ApiGet(paramInt(ctx, "id"))
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, m)
}

// handleApiCreate 新增接口。
// @Summary 新增接口
// @Tags system
// @Accept json
// @Produce json
// @Param body body dto.ApiSaveReq true "请求体"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/sys-api [post]
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
// @Summary 修改接口
// @Tags system
// @Accept json
// @Produce json
// @Param id path int true "接口ID"
// @Param body body dto.ApiSaveReq true "请求体"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/sys-api/{id} [put]
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
// @Summary 批量删除接口
// @Tags system
// @Accept json
// @Produce json
// @Param body body dto.IdsReq true "批量删除ID"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/sys-api [delete]
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

// handleCaptcha 返回图形验证码。
// @Summary 获取图形验证码
// @Tags system
// @Produce json
// @Success 200 {object} dto.CaptchaResp "验证码 ID 与图片地址"
// @Router /api/v1/captcha [get]
func handleCaptcha(ctx http.Context) error {
	id, uri := logic.Captcha()
	return ctx.Status(fiber.StatusOK).JSON(dto.CaptchaResp{Code: 200, Msg: "ok", Id: id, Data: uri})
}

// handleLogin 账号密码 + 图形验证码登录。
// @Summary 后台登录
// @Tags system
// @Accept json
// @Produce json
// @Param body body dto.AdminLoginReq true "登录参数"
// @Success 200 {object} dto.TokenResp "登录成功，token 与信封同级"
// @Router /api/v1/login [post]
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
// @Summary 注销登录
// @Tags system
// @Produce json
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/logout [post]
func handleLogout(ctx http.Context) error { return errs.OKMsg(ctx, "注销成功") }

// handleGetInfo 返回当前登录用户的角色、权限与资料。
// @Summary 获取当前用户信息
// @Tags system
// @Produce json
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/getinfo [get]
func handleGetInfo(ctx http.Context) error {
	return errs.OK(ctx, logic.GetInfo(global.CurrentUser(ctx), global.CurrentRole(ctx)))
}

// handleMenuRole 当前用户的动态路由菜单树。
// @Summary 获取当前用户动态菜单
// @Tags system
// @Produce json
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/menurole [get]
func handleMenuRole(ctx http.Context) error {
	menus, err := logic.MenuRole(global.CurrentRole(ctx))
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, menus)
}

// handleAppConfig 登录页/布局使用的品牌配置（无需鉴权）。
// @Summary 获取前端品牌配置
// @Tags system
// @Produce json
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/app-config [get]
func handleAppConfig(ctx http.Context) error {
	configs, err := logic.FrontendConfigs()
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, configs)
}

// handleConfigList 系统配置分页列表。
// @Summary 配置列表
// @Tags system
// @Produce json
// @Param pageIndex query int false "页码" default(1)
// @Param pageSize query int false "每页条数" default(10)
// @Param configName query string false "配置名称"
// @Param configKey query string false "配置键"
// @Param configType query string false "系统内置（Y/N）"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/config [get]
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
// @Summary 配置详情
// @Tags system
// @Produce json
// @Param id path int true "配置ID"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/config/{id} [get]
func handleConfigGet(ctx http.Context) error {
	m, err := logic.ConfigGet(paramInt(ctx, "id"))
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, m)
}

// handleConfigKey 按配置键取值。
// @Summary 按配置键取值
// @Tags system
// @Produce json
// @Param key path string true "配置键"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/configKey/{key} [get]
func handleConfigKey(ctx http.Context) error {
	m, err := logic.ConfigGetByKey(ctx.Params("key"))
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, dto.ConfigValueResp{ConfigValue: m.ConfigValue})
}

// handleConfigCreate 新增配置。
// @Summary 新增配置
// @Tags system
// @Accept json
// @Produce json
// @Param body body dto.ConfigSaveReq true "请求体"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/config [post]
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
// @Summary 修改配置
// @Tags system
// @Accept json
// @Produce json
// @Param id path int true "配置ID"
// @Param body body dto.ConfigSaveReq true "请求体"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/config/{id} [put]
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
// @Summary 批量删除配置
// @Tags system
// @Accept json
// @Produce json
// @Param body body dto.IdsReq true "批量删除ID"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/config [delete]
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
// @Summary 获取内置系统配置
// @Tags system
// @Produce json
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/set-config [get]
func handleGetSetConfig(ctx http.Context) error {
	values, err := logic.SettingValues()
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, values)
}

// handleUpdateSetConfig 批量保存系统配置页的内置键。
// @Summary 保存内置系统配置
// @Tags system
// @Accept json
// @Produce json
// @Param body body dto.SetConfigReq true "请求体"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/set-config [put]
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

// handleDictTypeList 字典类型分页列表。
// @Summary 字典类型列表
// @Tags system
// @Produce json
// @Param pageIndex query int false "页码" default(1)
// @Param pageSize query int false "每页条数" default(10)
// @Param dictName query string false "字典名称"
// @Param dictType query string false "字典类型"
// @Param status query string false "状态"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/dict/type [get]
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
// @Summary 字典类型详情
// @Tags system
// @Produce json
// @Param id path int true "字典类型ID"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/dict/type/{id} [get]
func handleDictTypeGet(ctx http.Context) error {
	m, err := logic.DictTypeGet(paramInt(ctx, "id"))
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, m)
}

// handleDictTypeCreate 新增字典类型。
// @Summary 新增字典类型
// @Tags system
// @Accept json
// @Produce json
// @Param body body dto.DictTypeSaveReq true "请求体"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/dict/type [post]
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
// @Summary 修改字典类型
// @Tags system
// @Accept json
// @Produce json
// @Param id path int true "字典类型ID"
// @Param body body dto.DictTypeSaveReq true "请求体"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/dict/type/{id} [put]
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
// @Summary 批量删除字典类型
// @Tags system
// @Accept json
// @Produce json
// @Param body body dto.IdsReq true "批量删除ID"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/dict/type [delete]
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
// @Summary 字典类型下拉项
// @Tags system
// @Produce json
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/dict/type-option-select [get]
func handleDictTypeOptions(ctx http.Context) error {
	options, err := logic.DictTypeOptions()
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, options)
}

// handleDictDataList 字典项分页列表。
// @Summary 字典项列表
// @Tags system
// @Produce json
// @Param pageIndex query int false "页码" default(1)
// @Param pageSize query int false "每页条数" default(10)
// @Param dictType query string false "字典类型"
// @Param dictLabel query string false "字典标签"
// @Param status query string false "状态"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/dict/data [get]
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
// @Summary 字典项详情
// @Tags system
// @Produce json
// @Param dictCode path int true "字典编码"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/dict/data/{dictCode} [get]
func handleDictDataGet(ctx http.Context) error {
	m, err := logic.DictDataGet(paramInt(ctx, "dictCode"))
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, m)
}

// handleDictDataCreate 新增字典项。
// @Summary 新增字典项
// @Tags system
// @Accept json
// @Produce json
// @Param body body dto.DictDataSaveReq true "请求体"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/dict/data [post]
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
// @Summary 修改字典项
// @Tags system
// @Accept json
// @Produce json
// @Param dictCode path int true "字典编码"
// @Param body body dto.DictDataSaveReq true "请求体"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/dict/data/{dictCode} [put]
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
// @Summary 批量删除字典项
// @Tags system
// @Accept json
// @Produce json
// @Param body body dto.IdsReq true "批量删除ID"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/dict/data [delete]
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
// @Summary 字典项下拉项
// @Tags system
// @Produce json
// @Param dictType query string true "字典类型"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/dict-data/option-select [get]
func handleDictDataOptions(ctx http.Context) error {
	options, err := logic.DictDataOptions(ctx.Query("dictType"))
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, options)
}

// handleLoginLogList 登录日志分页列表。
// @Summary 登录日志列表
// @Tags system
// @Produce json
// @Param pageIndex query int false "页码" default(1)
// @Param pageSize query int false "每页条数" default(10)
// @Param username query string false "登录账号"
// @Param ipaddr query string false "登录IP"
// @Param status query string false "登录状态"
// @Param beginTime query string false "开始时间"
// @Param endTime query string false "结束时间"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/sys-login-log [get]
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
// @Summary 登录日志详情
// @Tags system
// @Produce json
// @Param id path int true "日志ID"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/sys-login-log/{id} [get]
func handleLoginLogGet(ctx http.Context) error {
	m, err := logic.LoginLogGet(paramInt(ctx, "id"))
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, m)
}

// handleLoginLogDelete 批量删除登录日志。
// @Summary 批量删除登录日志
// @Tags system
// @Accept json
// @Produce json
// @Param body body dto.IdsReq true "批量删除ID"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/sys-login-log [delete]
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
// @Summary 操作日志列表
// @Tags system
// @Produce json
// @Param pageIndex query int false "页码" default(1)
// @Param pageSize query int false "每页条数" default(10)
// @Param title query string false "模块标题"
// @Param operName query string false "操作人"
// @Param status query string false "操作状态"
// @Param beginTime query string false "开始时间"
// @Param endTime query string false "结束时间"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/sys-opera-log [get]
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
// @Summary 批量删除操作日志
// @Tags system
// @Accept json
// @Produce json
// @Param body body dto.IdsReq true "批量删除ID"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/sys-opera-log [delete]
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
// @Summary 清空操作日志
// @Tags system
// @Produce json
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/operlog/clean [delete]
func handleOperaLogClean(ctx http.Context) error {
	if err := logic.OperaLogClean(); err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OKMsg(ctx, "清空成功")
}

// handleMenuList 菜单树（不分页）。
// @Summary 菜单树
// @Tags system
// @Produce json
// @Param title query string false "菜单标题"
// @Param menuName query string false "菜单名称"
// @Param visible query string false "显示状态"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/menu [get]
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
// @Summary 菜单详情
// @Tags system
// @Produce json
// @Param menuId path int true "菜单ID"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/menu/{menuId} [get]
func handleMenuGet(ctx http.Context) error {
	m, err := logic.MenuGet(paramInt(ctx, "menuId"))
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, m)
}

// handleRoleMenuTreeselect 角色授权树。
// @Summary 角色授权菜单树
// @Tags system
// @Produce json
// @Param roleId path int true "角色ID"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/roleMenuTreeselect/{roleId} [get]
func handleRoleMenuTreeselect(ctx http.Context) error {
	resp, err := logic.RoleMenuTreeselect(paramInt(ctx, "roleId"))
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, resp)
}

// handleMenuCreate 新增菜单。
// @Summary 新增菜单
// @Tags system
// @Accept json
// @Produce json
// @Param body body dto.MenuSaveReq true "请求体"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/menu [post]
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
// @Summary 修改菜单
// @Tags system
// @Accept json
// @Produce json
// @Param id path int true "菜单ID"
// @Param body body dto.MenuSaveReq true "请求体"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/menu/{id} [put]
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
// @Summary 批量删除菜单
// @Tags system
// @Accept json
// @Produce json
// @Param body body dto.IdsReq true "批量删除ID"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/menu [delete]
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

// handleRoleList 角色分页列表。
// @Summary 角色列表
// @Tags system
// @Produce json
// @Param pageIndex query int false "页码" default(1)
// @Param pageSize query int false "每页条数" default(10)
// @Param roleName query string false "角色名"
// @Param roleKey query string false "权限字符"
// @Param status query string false "状态"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/role [get]
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
// @Summary 角色详情
// @Tags system
// @Produce json
// @Param roleId path int true "角色ID"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/role/{roleId} [get]
func handleRoleGet(ctx http.Context) error {
	m, err := logic.RoleGet(paramInt(ctx, "roleId"))
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, m)
}

// handleRoleCreate 新增角色。
// @Summary 新增角色
// @Tags system
// @Accept json
// @Produce json
// @Param body body dto.RoleSaveReq true "请求体"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/role [post]
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
// @Summary 修改角色
// @Tags system
// @Accept json
// @Produce json
// @Param roleId path int true "角色ID"
// @Param body body dto.RoleSaveReq true "请求体"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/role/{roleId} [put]
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
// @Summary 批量删除角色
// @Tags system
// @Accept json
// @Produce json
// @Param body body dto.IdsReq true "批量删除ID"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/role [delete]
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
// @Summary 启用/停用角色
// @Tags system
// @Accept json
// @Produce json
// @Param body body dto.RoleStatusReq true "请求体"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/role-status [put]
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

// handleUserList 用户分页列表。
// @Summary 用户列表
// @Tags system
// @Produce json
// @Param pageIndex query int false "页码" default(1)
// @Param pageSize query int false "每页条数" default(10)
// @Param username query string false "用户名"
// @Param phone query string false "手机号"
// @Param status query string false "状态"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/sys-user [get]
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
// @Summary 用户详情
// @Tags system
// @Produce json
// @Param userId path int true "用户ID"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/sys-user/{userId} [get]
func handleUserGet(ctx http.Context) error {
	m, err := logic.UserGet(paramInt(ctx, "userId"))
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, m)
}

// handleUserCreate 新增用户。
// @Summary 新增用户
// @Tags system
// @Accept json
// @Produce json
// @Param body body dto.UserSaveReq true "请求体"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/sys-user [post]
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
// @Summary 修改用户
// @Tags system
// @Accept json
// @Produce json
// @Param body body dto.UserSaveReq true "请求体"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/sys-user [put]
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
// @Summary 批量删除用户
// @Tags system
// @Accept json
// @Produce json
// @Param body body dto.IdsReq true "批量删除ID"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/sys-user [delete]
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
// @Summary 启用/停用用户
// @Tags system
// @Accept json
// @Produce json
// @Param body body dto.UserStatusReq true "请求体"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/user/status [put]
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
// @Summary 重置用户密码
// @Tags system
// @Accept json
// @Produce json
// @Param body body dto.UserPasswordResetReq true "请求体"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/user/pwd/reset [put]
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
// @Summary 获取当前用户资料
// @Tags system
// @Produce json
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/user/profile [get]
func handleUserProfile(ctx http.Context) error {
	roleName := ""
	if role := global.CurrentRole(ctx); role != nil {
		roleName = role.RoleName
	}
	return errs.OK(ctx, dto.UserProfileResp{User: global.CurrentUser(ctx), RoleName: roleName})
}

// handleUserProfileUpdate 修改当前登录用户的资料。
// @Summary 修改当前用户资料
// @Tags system
// @Accept json
// @Produce json
// @Param body body dto.UserProfileUpdateReq true "请求体"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/sys-user/profile [put]
func handleUserProfileUpdate(ctx http.Context) error {
	var req dto.UserProfileUpdateReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	m, err := logic.UserUpdateProfile(global.CurrentUser(ctx).UserId, req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, m)
}

// handleUserPwdSet 当前登录用户修改密码。
// @Summary 修改当前用户密码
// @Tags system
// @Accept json
// @Produce json
// @Param body body dto.UserPasswordSetReq true "请求体"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/user/pwd/set [put]
func handleUserPwdSet(ctx http.Context) error {
	var req dto.UserPasswordSetReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	if err := logic.UserSetPassword(global.CurrentUser(ctx), req); err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OKMsg(ctx, "密码修改成功")
}
