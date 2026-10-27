// Package global 后台全局公共件：初始化装配、鉴权/审计中间件与当前用户上下文读取。
package global

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"

	"server-cluster/mesh/admin/app/logic"
	"server-cluster/mesh/admin/app/repo"
	"server-cluster/share/errs"
	"server-cluster/share/model"
)

const (
	localUser = "admin.user"
	localRole = "admin.role"
)

// alwaysAllowed 登录后即可访问、不参与接口授权的端点。
var alwaysAllowed = map[string]struct{}{
	"/api/v1/logout":           {},
	"/api/v1/getinfo":          {},
	"/api/v1/menurole":         {},
	"/api/v1/user/profile":     {},
	"/api/v1/sys-user/profile": {},
	"/api/v1/user/pwd/set":     {},
}

// publicPaths 无需登录即可访问的端点。
var publicPaths = map[string]struct{}{
	"/api/v1/captcha":    {},
	"/api/v1/login":      {},
	"/api/v1/app-config": {},
}

// isPublic 判断端点是否公开。
func isPublic(path string) bool {
	_, ok := publicPaths[path]
	return ok
}

// CORSMiddleware 跨域放行（前端与后端分离部署/不同端口，预检请求在这里直接应答）。
func CORSMiddleware() fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
	})
}

// RequireDB 数据库未配置时直接拒绝请求，避免空句柄穿透到业务层。
func RequireDB(c fiber.Ctx) error {
	if !repo.R().Available() {
		return errs.FailCtx(c, 500, "数据库未配置")
	}
	return c.Next()
}

// AuthMiddleware 校验 Authorization Bearer 令牌并注入当前用户/角色。
func AuthMiddleware(c fiber.Ctx) error {
	if isPublic(c.Path()) {
		return c.Next()
	}
	token := strings.TrimSpace(strings.TrimPrefix(c.Get("Authorization"), "Bearer "))
	if token == "" {
		return errs.FailCtx(c, 401, "未登录或登录已过期")
	}
	user, role, err := logic.Authenticate(token)
	if err != nil {
		return errs.FailCtxErr(c, err)
	}
	c.Locals(localUser, user)
	c.Locals(localRole, role)
	return c.Next()
}

// PermissionMiddleware 按 sys_casbin_rule 校验接口访问权限；roleKey=admin 全放行。
func PermissionMiddleware(c fiber.Ctx) error {
	if isPublic(c.Path()) {
		return c.Next()
	}
	role, ok := c.Locals(localRole).(*model.SysRole)
	if !ok {
		return errs.FailCtx(c, 401, "未登录或登录已过期")
	}
	if _, skip := alwaysAllowed[c.Path()]; skip {
		return c.Next()
	}
	allowed, err := logic.Allow(role.RoleKey, c.Path(), c.Method())
	if err != nil {
		return errs.FailCtx(c, 500, "权限校验失败")
	}
	if !allowed {
		return errs.FailCtx(c, 403, "无接口访问权限")
	}
	return c.Next()
}

// OperaLogMiddleware 记录写操作审计；挂在校验之后执行，因此能拿到当前用户。
func OperaLogMiddleware(c fiber.Ctx) error {
	if c.Method() == fiber.MethodGet || isPublic(c.Path()) {
		return c.Next()
	}
	start := time.Now()
	nextErr := c.Next()
	entry := &model.SysOperaLog{
		Title:         businessTitle(c.Method()) + " " + c.Path(),
		BusinessType:  businessType(c.Method()),
		RequestMethod: c.Method(),
		OperUrl:       c.Path(),
		OperIp:        c.IP(),
		Status:        "2",
		OperTime:      start,
		OperLocation:  "内网",
		OperParam:     truncate(string(c.Body()), 2000),
		JsonResult:    truncate(string(c.Response().Body()), 2000),
		LatencyTime:   fmt.Sprintf("%dms", time.Since(start).Milliseconds()),
	}
	if user := CurrentUser(c); user != nil {
		entry.OperName = user.Username
	}
	if nextErr != nil || responseCode(c) != 200 {
		entry.Status = "1"
	}
	logic.RecordOpera(entry)
	return nextErr
}

// CurrentUser 取当前登录用户。
func CurrentUser(c fiber.Ctx) *model.SysUser {
	user, _ := c.Locals(localUser).(*model.SysUser)
	return user
}

// CurrentRole 取当前登录角色。
func CurrentRole(c fiber.Ctx) *model.SysRole {
	role, _ := c.Locals(localRole).(*model.SysRole)
	return role
}

// responseCode 读取响应信封中的业务码。
func responseCode(c fiber.Ctx) int {
	var envelope struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(c.Response().Body(), &envelope); err != nil {
		return 200
	}
	return envelope.Code
}

// businessType 由请求方法推导业务类型。
func businessType(method string) string {
	switch method {
	case fiber.MethodPost:
		return "1"
	case fiber.MethodPut:
		return "2"
	case fiber.MethodDelete:
		return "3"
	default:
		return "0"
	}
}

// businessTitle 由请求方法推导业务标题。
func businessTitle(method string) string {
	switch method {
	case fiber.MethodPost:
		return "新增"
	case fiber.MethodPut:
		return "修改"
	case fiber.MethodDelete:
		return "删除"
	default:
		return "操作"
	}
}

// truncate 截断超长文本，避免审计表被大字段撑爆。
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
