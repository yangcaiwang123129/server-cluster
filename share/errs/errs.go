// Package errs 统一错误出口：handler/middleware 层只有一个 Respond(ctx, err, data)。
// 失败识别顺序：codes.Code → gorm.ErrRecordNotFound → Duplicate entry → 兜底 500 + 日志。
// 成功识别：err == nil。
//
// 错误码定义见 errs_code.go（与 pb.ErrCode 数值一致，客户端同用）。
package errs

import (
	"errors"
	"log"
	"server-cluster/share/dto"
	"server-cluster/share/util"
	"strings"

	"github.com/dobyte/due/component/http/v2"
	"github.com/dobyte/due/v2/codes"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// ============ HTTP 语义码（admin 侧专用，不进 pb） ============

const (
	HTTPStatusOK     = 200
	HTTPBadRequest   = 400
	HTTPUnauthorized = 401
	HTTPForbidden    = 403
	HTTPNotFound     = 404
	HTTPInternal     = 500
)

// ============ Respond：handler 层唯一出口（http.Context，due 框架封装） ============

// Respond 统一响应出口（handler 层用 due http.Context）。
//
//	err == nil                 → 200 + Resp{code:200, msg:"ok"[或successMsg], data:data}
//	codes.Convert(err) != nil → 200 + Resp{code:code.Code(), msg:code.Message()}
//	gorm.ErrRecordNotFound    → 200 + Resp{code:404, msg:"记录不存在"}
//	Duplicate entry           → 200 + Resp{code:400, msg:"数据已存在"}
//	普通 error                → 200 + Resp{code:500, msg:"internal server error"}
//	                             + log.Printf 记录原始错误（不暴露给前端）
//
// 用法：
//
//	return errs.Respond(ctx, err, data)                     // 成功 msg="ok"
//	return errs.Respond(ctx, err, nil, "删除成功")          // 成功 msg="删除成功"
func Respond(ctx http.Context, err error, data any, successMsg ...string) error {
	if err == nil {
		msg := "ok"
		if len(successMsg) > 0 {
			msg = successMsg[0]
		}
		return ctx.Status(fiber.StatusOK).JSON(dto.Resp{Code: HTTPStatusOK, Msg: msg, Data: data})
	}
	code, msg := classify(err)
	return ctx.Status(fiber.StatusOK).JSON(dto.Resp{Code: code, Msg: msg})
}

// ============ RespondCtx：middleware 层 fiber.Ctx 版本 ============

// RespondCtx fiber.Ctx 版统一响应出口（middleware 专用）。
func RespondCtx(c fiber.Ctx, err error, data any, successMsg ...string) error {
	if err == nil {
		msg := "ok"
		if len(successMsg) > 0 {
			msg = successMsg[0]
		}
		return c.Status(fiber.StatusOK).JSON(dto.Resp{Code: HTTPStatusOK, Msg: msg, Data: data})
	}
	code, msg := classify(err)
	return c.Status(fiber.StatusOK).JSON(dto.Resp{Code: code, Msg: msg})
}

// classify 统一错误分类：返回 (code, msg)。
func classify(err error) (int, string) {
	if c := codes.Convert(err); c != nil {
		return c.Code(), c.Message()
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return HTTPNotFound, "记录不存在"
	}
	if strings.Contains(err.Error(), "Duplicate entry") {
		return HTTPBadRequest, "数据已存在，请检查唯一字段"
	}
	log.Printf("[ERRS] %s", err.Error())
	return HTTPInternal, "internal server error"
}

// ============ 便捷函数（薄封装，内部统一走 Respond/RespondCtx + classify） ============
//
// 这 6 个函数是 handler/middleware 层的"语义糖"，内部全部委托给 Respond/RespondCtx，
// 保证错误分类逻辑只有 classify() 一处。新代码鼓励直接用 Respond/RespondCtx。

// OK 成功响应（due http.Context 版）。
func OK(ctx http.Context, data any) error { return Respond(ctx, nil, data) }

// OKMsg 成功响应带自定义消息（due http.Context 版，无 data）。
func OKMsg(ctx http.Context, msg string) error { return Respond(ctx, nil, nil, msg) }

// Fail 失败响应，硬编码 HTTP 语义码 + 自定义消息（due http.Context 版）。
func Fail(ctx http.Context, httpCode int, msg string) error {
	return Respond(ctx, codes.NewCode(httpCode, msg).Err(), nil)
}

// FailErr 失败响应，直接透传 error（由 classify 自动分类）（due http.Context 版）。
func FailErr(ctx http.Context, err error) error { return Respond(ctx, err, nil) }

// FailCtx 失败响应，硬编码 HTTP 语义码 + 自定义消息（fiber.Ctx 版，middleware 专用）。
func FailCtx(c fiber.Ctx, httpCode int, msg string) error {
	return RespondCtx(c, codes.NewCode(httpCode, msg).Err(), nil)
}

// FailCtxErr 失败响应，直接透传 error（fiber.Ctx 版，middleware 专用）。
func FailCtxErr(c fiber.Ctx, err error) error { return RespondCtx(c, err, nil) }

// NewError 创建一个 codes.Code error（logic 层通用入口，HTTP 码或 pb 业务码都能收）。
func NewError(code int, msg string) error { return codes.NewCode(code, msg).Err() }

// ============ Page：分页组装 ============

// Page 组装分页 data（纯数据，HTTP 响应由 Respond 负责）。
func Page[T any](list []T, total int64, pageIndex, pageSize int) util.PageResp[T] {
	return util.PageResp[T]{List: list, Count: total, PageIndex: pageIndex, PageSize: pageSize}
}
