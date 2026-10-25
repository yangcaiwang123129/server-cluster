// Package web HTTP 层：向 due http 组件注册业务路由（swagger 由 etc.http.swagger 配置驱动）。
package web

import (
	"github.com/dobyte/due/component/http/v2"

	"server-cluster/mesh/login/app/web/handler"
)

// Register 向 due http 组件注册业务路由。
func Register(router http.Router) {
	api := router.Group("/api/v1")
	api.Post("/account/register", handler.Register)
	api.Post("/account/login", handler.Login)
	api.Get("/account/servers", handler.ServerList)
}
