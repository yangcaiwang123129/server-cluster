// Package global 后台全局公共件：初始化装配、鉴权/审计中间件与当前用户上下文读取。
package global

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"gorm.io/gorm"

	"server-cluster/internal/svc"
	"server-cluster/share/consts"
	"server-cluster/share/errs"
	"github.com/dobyte/due/v2/codes"
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
	if _, err := svc.Ctx().DB(); err != nil {
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
	user, role, err := Authenticate(token)
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
	allowed, err := Allow(role.RoleKey, c.Path(), c.Method())
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
		Status:        consts.LogStatusSuccess,
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
		entry.Status = consts.LogStatusFail
	}
	RecordOpera(entry)
	return nextErr
}

// Authenticate 解析令牌并加载用户与角色，供鉴权中间件使用。
// 查询逻辑与 repo 中的 GetUser/RoleForUser 保持一致，此处内联以避免 middleware 依赖 repo。
func Authenticate(token string) (*model.SysUser, *model.SysRole, error) {
	payload, err := svc.JWT().ExtractPayload(token)
	if err != nil {
		return nil, nil, codes.NewCode(401, "未登录或登录已过期").Err()
	}
	uid, ok := payload["uid"].(float64)
	if !ok {
		return nil, nil, codes.NewCode(401, "令牌无效").Err()
	}
	d, err := svc.Ctx().DB()
	if err != nil {
		return nil, nil, codes.NewCode(500, "数据库未配置").Err()
	}
	var user model.SysUser
	if err := d.First(&user, "user_id = ?", int(uid)).Error; err != nil {
		return nil, nil, codes.NewCode(401, "账号不存在").Err()
	}
	if user.Status != consts.StatusEnabled {
		return nil, nil, codes.NewCode(403, "账号已停用").Err()
	}
	var role model.SysRole
	if err := d.First(&role, "role_id = ?", user.RoleId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, codes.NewCode(403, "账号未绑定角色").Err()
		}
		return nil, nil, codes.NewCode(500, "查询角色失败").Err()
	}
	if role.Status != consts.StatusEnabled {
		return nil, nil, codes.NewCode(403, "角色已停用").Err()
	}
	return &user, &role, nil
}

// Allow 判断角色是否可访问指定接口；角色标识为 admin 时全部放行。
// 规则中的路径是路由模板（如 /api/v1/sys-user/:userId），需按模板匹配实际请求路径。
func Allow(roleKey, path, method string) (bool, error) {
	if roleKey == consts.AdminRoleKey {
		return true, nil
	}
	d, err := svc.Ctx().DB()
	if err != nil {
		return false, err
	}
	var rules []model.SysCasbinRule
	if err := d.
		Where("ptype = ? AND v0 = ? AND v2 = ?", "p", roleKey, method).
		Find(&rules).Error; err != nil {
		return false, err
	}
	for _, rule := range rules {
		if matchRulePath(rule.V1, path) {
			return true, nil
		}
	}
	return false, nil
}

// matchRulePath 按路由模板匹配实际请求路径，模板中的 :param / * 段匹配任意单段。
func matchRulePath(pattern, path string) bool {
	if pattern == path {
		return true
	}
	ps := strings.Split(strings.Trim(pattern, "/"), "/")
	as := strings.Split(strings.Trim(path, "/"), "/")
	if len(ps) != len(as) {
		return false
	}
	for i, p := range ps {
		if strings.HasPrefix(p, ":") || strings.HasPrefix(p, "*") {
			continue
		}
		if p != as[i] {
			return false
		}
	}
	return true
}

// RecordOpera 写入操作审计；失败不影响主流程。
func RecordOpera(m *model.SysOperaLog) {
	d, err := svc.Ctx().DB()
	if err != nil {
		return
	}
	_ = d.Create(m).Error
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
