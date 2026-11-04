package handler

import (
	"server-cluster/share/errs"

	"github.com/dobyte/due/component/http/v2"
)

// handle 通用 handler 泛型包装：业务函数只关心取参与调 logic，返回 (数据, error)。
// err 非空走 errs.FailErr（自动分类业务码），成功走 errs.OK(data)。
// 业务内需要失败时返回零值 + errs.NewError(code, msg)，与原 errs.Fail 等价。
func handle[R any](fn func(ctx http.Context) (R, error)) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		resp, err := fn(ctx)
		if err != nil {
			return errs.FailErr(ctx, err)
		}
		return errs.OK(ctx, resp)
	}
}

// handleMsg 同 handle，成功时额外携带自定义提示文案（如「删除成功」）。
func handleMsg[R any](msg string, fn func(ctx http.Context) (R, error)) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		resp, err := fn(ctx)
		if err != nil {
			return errs.FailErr(ctx, err)
		}
		return errs.OK(ctx, resp, msg)
	}
}

// 路由注册助手：直接接收 (数据, error) 业务函数，路由行不再出现 handle 包装。
// 返回值类型由业务函数推断；无数据的操作（删除/状态变更）用 (any, error) 返回 nil。
// 特殊流程（captcha/login 等自定义响应）不走助手，直接注册 func(ctx) error。

// GET 注册 GET 路由。
func GET[R any](g http.Router, path string, fn func(ctx http.Context) (R, error)) {
	g.Get(path, handle(fn))
}

// POST 注册 POST 路由。
func POST[R any](g http.Router, path string, fn func(ctx http.Context) (R, error)) {
	g.Post(path, handle(fn))
}

// PUT 注册 PUT 路由。
func PUT[R any](g http.Router, path string, fn func(ctx http.Context) (R, error)) {
	g.Put(path, handle(fn))
}

// DELETE 注册 DELETE 路由。
func DELETE[R any](g http.Router, path string, fn func(ctx http.Context) (R, error)) {
	g.Delete(path, handle(fn))
}

// POSTMsg 注册 POST 路由，成功响应携带自定义提示文案。
func POSTMsg[R any](g http.Router, path, msg string, fn func(ctx http.Context) (R, error)) {
	g.Post(path, handleMsg(msg, fn))
}

// PUTMsg 注册 PUT 路由，成功响应携带自定义提示文案。
func PUTMsg[R any](g http.Router, path, msg string, fn func(ctx http.Context) (R, error)) {
	g.Put(path, handleMsg(msg, fn))
}

// DELETEMsg 注册 DELETE 路由，成功响应携带自定义提示文案。
func DELETEMsg[R any](g http.Router, path, msg string, fn func(ctx http.Context) (R, error)) {
	g.Delete(path, handleMsg(msg, fn))
}
