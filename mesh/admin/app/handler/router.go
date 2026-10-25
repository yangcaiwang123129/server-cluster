// Package handler HTTP 层：每个模块一个文件，内含路由注册与处理函数。
package handler

import (
	"github.com/dobyte/due/component/http/v2"

	"server-cluster/mesh/admin/app/globol"
)

// Register 注册管理后台路由：公开组与鉴权组各建一次，模块各自挂载。
func Register(router http.Router) {
	public := router.Group("/api/v1", globol.RequireDB)
	// 审计最先进入、最后退出，因此能拿到鉴权后的用户。
	secure := router.Group("/api/v1", globol.RequireDB, globol.OperaLogMiddleware, globol.AuthMiddleware, globol.PermissionMiddleware)

	registerAuth(public, secure)
	registerUser(secure)
	registerRole(secure)
	registerMenu(secure)
	registerApi(secure)
	registerDict(secure)
	registerConfig(secure)
	registerLog(secure)
}
