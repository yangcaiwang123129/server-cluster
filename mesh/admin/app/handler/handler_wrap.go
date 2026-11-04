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
