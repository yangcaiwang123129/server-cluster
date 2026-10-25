// Package handler HTTP 层：每个模块一个文件，内含路由注册与处理函数。
package handler

import (
	"github.com/dobyte/due/component/http/v2"

	"server-cluster/mesh/admin/app/globol"
)

// Register 注册管理后台路由。
//
// Fiber 的 Group 中间件按前缀对所有 /api/v1 请求生效，因此只建一个组：
// 公开端点（验证码/登录/品牌配置）由中间件按路径放行。
func Register(router http.Router) {
	group := router.Group("/api/v1",
		globol.CORSMiddleware(),
		globol.RequireDB,
		globol.OperaLogMiddleware,
		globol.AuthMiddleware,
		globol.PermissionMiddleware,
	)

	registerAuth(group)
	registerUser(group)
	registerRole(group)
	registerMenu(group)
	registerApi(group)
	registerDict(group)
	registerConfig(group)
	registerLog(group)
}
