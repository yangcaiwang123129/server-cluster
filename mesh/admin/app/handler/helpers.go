package handler

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/dobyte/due/component/http/v2"

	"server-cluster/mesh/admin/app/global"
)

// authMiddlewares 所有受保护路由共用的中间件链：CORS + DB + 操作日志 + 鉴权 + 权限。
var authMiddlewares = []any{
	global.CORSMiddleware(),
	global.RequireDB,
	global.OperaLogMiddleware,
	global.AuthMiddleware,
	global.PermissionMiddleware,
}

// protectedGroup 创建受保护路由组（需鉴权 + DB + 权限），自动带上 /api/v1 前缀。
func protectedGroup(router http.Router) http.Router {
	return router.Group("/api/v1", authMiddlewares...)
}

// gmGroup 创建 GM 路由组（/api/v1/gm），所有游戏内操作、模拟器、客户端版本都挂在这里。
func gmGroup(router http.Router) http.Router {
	return router.Group("/api/v1/gm", authMiddlewares...)
}

// publicGroup 创建公开路由组（仅 CORS + DB，不需要鉴权），带 /api/v1 前缀。
func publicGroup(router http.Router) http.Router {
	return router.Group("/api/v1",
		global.CORSMiddleware(),
		global.RequireDB,
	)
}

// bindBody 解析 JSON 请求体。
func bindBody(ctx http.Context, v any) error {
	return json.Unmarshal(ctx.Body(), v)
}

// pageParams 读取分页参数。
func pageParams(ctx http.Context) (pageIndex, pageSize int) {
	pageIndex, _ = strconv.Atoi(ctx.Query("pageIndex"))
	pageSize, _ = strconv.Atoi(ctx.Query("pageSize"))
	return
}

// paramInt 读取路径参数中的整数。
func paramInt(ctx http.Context, name string) int {
	v, _ := strconv.Atoi(ctx.Params(name))
	return v
}

// queryPlayerID 读取查询串中的玩家ID；非法或缺省返回 0。
func queryPlayerID(ctx http.Context) int64 {
	playerID, _ := strconv.ParseInt(ctx.Query("playerId"), 10, 64)
	if playerID <= 0 {
		return 0
	}
	return playerID
}

// orderBy 解析 <prop>Order 排序参数，只接受白名单列。
func orderBy(ctx http.Context, columns map[string]string) string {
	for prop, column := range columns {
		switch strings.ToLower(ctx.Query(prop + "Order")) {
		case "asc":
			return column + " asc"
		case "desc":
			return column + " desc"
		}
	}
	return ""
}
