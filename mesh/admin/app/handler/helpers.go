package handler

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/dobyte/due/component/http/v2"
)

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

// orderBy 解析 useTable 的 <prop>Order 排序参数，只接受白名单列。
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
