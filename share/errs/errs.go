// Package errs 公共错误定义：基于 consts 中的错误码预生成可直接返回的 error。
package errs

import (
	"errors"
	"server-cluster/share/dto"
	"server-cluster/share/util"
	"strings"

	"github.com/dobyte/due/component/http/v2"
	"github.com/dobyte/due/v2/codes"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

var (
	ErrInternal       = CodeInternalError.Err()
	ErrInvalidParams  = CodeInvalidParams.Err()
	ErrUnauthorized   = CodeUnauthorized.Err()
	ErrPlayerNotFound = CodePlayerNotFound.Err()
	ErrPlayerExists   = CodePlayerExists.Err()
	ErrPasswordWrong  = CodePasswordWrong.Err()
	ErrTokenInvalid   = CodeTokenInvalid.Err()
)

// WithMessage 基于既有错误码改写提示文案，返回新的 error（不影响原码）。
func WithMessage(code *codes.Code, message string) error {
	return code.WithMessage(message).Err()
}

type Error struct {
	Code int
	Msg  string
}

// Error 实现 error 接口。
func (e *Error) Error() string { return e.Msg }

// NewError 构造业务错误。
func NewError(code int, msg string) *Error { return &Error{Code: code, Msg: msg} }

// OK 返回成功响应。
func OK(ctx http.Context, data any) error {
	return ctx.Status(fiber.StatusOK).JSON(dto.Resp{Code: 200, Msg: "ok", Data: data})
}

// OKMsg 返回带提示语的成功响应。
func OKMsg(ctx http.Context, msg string) error {
	return ctx.Status(fiber.StatusOK).JSON(dto.Resp{Code: 200, Msg: msg})
}

// Fail 返回业务失败响应；HTTP 状态固定 200，由 code 表达结果。
func Fail(ctx http.Context, code int, msg string) error {
	return ctx.Status(fiber.StatusOK).JSON(dto.Resp{Code: code, Msg: msg})
}

// FailCtx 供 fiber 原生中间件使用的失败响应。
func FailCtx(c fiber.Ctx, code int, msg string) error {
	return c.Status(fiber.StatusOK).JSON(dto.Resp{Code: code, Msg: msg})
}

// FailCtxErr 将业务/数据层错误映射为前端可读的失败响应。
func FailCtxErr(c fiber.Ctx, err error) error {
	var be *Error
	if errors.As(err, &be) {
		return FailCtx(c, be.Code, be.Msg)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return FailCtx(c, 404, "记录不存在")
	}
	if strings.Contains(err.Error(), "Duplicate entry") {
		return FailCtx(c, 400, "数据已存在，请检查唯一字段")
	}
	return FailCtx(c, 500, err.Error())
}

// FailErr 与 FailCtxErr 相同，供 handler 直接使用。
func FailErr(ctx http.Context, err error) error { return FailCtxErr(ctx, err) }

// Page 组装分页响应。
func Page[T any](list []T, total int64, pageIndex, pageSize int) util.PageResp[T] {
	return util.PageResp[T]{List: list, Count: total, PageIndex: pageIndex, PageSize: pageSize}
}
