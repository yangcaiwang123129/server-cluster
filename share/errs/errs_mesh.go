// Package errs 统一错误出口：admin HTTP 侧（http.Context / fiber.Ctx）与 node proto 侧共用一套错误码。
//
// 错误码体系（pb common.proto ErrCode 为唯一定义点，客户端同用）：
//
//	0     成功（= pb.ErrCode_ERR_CODE_OK）
//	1xxx  通用：1001 参数错误、1002 未登录/令牌失效
//	2xxx  玩家账号：2000-2003
//	3xxx  登录鉴权：3001-3004
//	4xxx  活动/任务：4001-4201
//	403   无接口访问权限（admin HTTP 语义补充，不进 pb）
//	500   内部错误兜底（admin HTTP 语义补充，不进 pb）
//
// node proto 侧 Result.Code 直接用 pb.ErrCode；admin HTTP 侧 Resp.Code 用同一套 int 值，
// 403/500 是仅 admin 侧使用的 HTTP 语义补充码。
package errs

import (
	"errors"
	"strings"

	"server-cluster/share/dto"
	"server-cluster/share/pb"
	"server-cluster/share/util"

	"github.com/dobyte/due/component/http/v2"
	"github.com/dobyte/due/v2/codes"
	"github.com/dobyte/due/v2/log"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// ============ admin HTTP Resp.code 统一取值 ============

const (
	// RespCodeOK 成功（= pb.ErrCode_ERR_CODE_OK）。
	RespCodeOK = 0
	// RespCodeForbidden 无接口访问权限（登录有效但 RBAC 拒绝，或账号/角色被停用）。
	RespCodeForbidden = 403
	// RespCodeInternal 内部错误兜底（未知错误、依赖故障、上游不可达）。
	RespCodeInternal = 500
)

// pb 业务码 int 形式（供 Fail/FailCtx/NewError 构造，避免到处写 int(pb.ErrCode_...)）。
const (
	CodeNumInvalidParams = int(pb.ErrCode_ERR_CODE_INVALID_PARAMS) // 1001 参数错误
	CodeNumUnauthorized  = int(pb.ErrCode_ERR_CODE_UNAUTHORIZED)   // 1002 未登录/令牌失效
)

// ============ Respond：handler 层唯一出口（http.Context，due 框架封装） ============

// Respond 统一响应出口（handler 层用 due http.Context）。
//
//	err == nil                 → Resp{code:0, msg:"ok"[或successMsg], data:data}
//	codes.Convert(err) != nil → Resp{code:业务码原样透传, msg:err消息}
//	                             （0/1001/1002/2xxx/3xxx/4xxx/403/500，由构造点保证合法性）
//	gorm.ErrRecordNotFound    → Resp{code:500, msg:"记录不存在"}
//	Duplicate entry           → Resp{code:500, msg:"数据已存在"}
//	普通 error                → Resp{code:500, msg:"internal server error"}
//	                             + log.Errorf 记录原始错误（不暴露给前端）
//
// 用法：
//
//	return errs.Respond(ctx, err, data)            // 成功 msg="ok"
//	return errs.Respond(ctx, err, nil, "删除成功")  // 成功 msg="删除成功"
func Respond(ctx http.Context, err error, data any, successMsg ...string) error {
	if err == nil {
		msg := "ok"
		if len(successMsg) > 0 {
			msg = successMsg[0]
		}
		return ctx.Status(fiber.StatusOK).JSON(dto.Resp{Code: RespCodeOK, Msg: msg, Data: data})
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
		return c.Status(fiber.StatusOK).JSON(dto.Resp{Code: RespCodeOK, Msg: msg, Data: data})
	}
	code, msg := classify(err)
	return c.Status(fiber.StatusOK).JSON(dto.Resp{Code: code, Msg: msg})
}

// classify 统一错误分类：返回 (业务码, msg)。
func classify(err error) (int, string) {
	if c := codes.Convert(err); c != nil {
		return c.Code(), c.Message()
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return RespCodeInternal, "记录不存在"
	}
	if strings.Contains(err.Error(), "Duplicate entry") {
		return RespCodeInternal, "数据已存在，请检查唯一字段"
	}
	log.Errorf("[ERRS] %s", err.Error())
	return RespCodeInternal, "internal server error"
}

// ============ 便捷函数（薄封装，内部统一走 Respond/RespondCtx + classify） ============

// OK 成功响应（due http.Context 版）。
func OK(ctx http.Context, data any) error { return Respond(ctx, nil, data) }

// OKMsg 成功响应带自定义消息（due http.Context 版，无 data）。
func OKMsg(ctx http.Context, msg string) error { return Respond(ctx, nil, nil, msg) }

// Fail 失败响应，手动指定业务码（pb 业务码或 403/500）（due http.Context 版）。
func Fail(ctx http.Context, code int, msg string) error {
	return Respond(ctx, codes.NewCode(code, msg).Err(), nil)
}

// FailParams 参数错误响应（pb 1001），handler 校验入参失败时的统一入口。
func FailParams(ctx http.Context, msg string) error {
	return Fail(ctx, CodeNumInvalidParams, msg)
}

// FailErr 失败响应，直接透传 error（由 classify 自动分类）（due http.Context 版）。
func FailErr(ctx http.Context, err error) error { return Respond(ctx, err, nil) }

// FailCtx 失败响应，手动指定业务码（fiber.Ctx 版，middleware 专用）。
func FailCtx(c fiber.Ctx, code int, msg string) error {
	return RespondCtx(c, codes.NewCode(code, msg).Err(), nil)
}

// FailCtxErr 失败响应，直接透传 error（fiber.Ctx 版，middleware 专用）。
func FailCtxErr(c fiber.Ctx, err error) error { return RespondCtx(c, err, nil) }

// NewError 创建一个 codes.Code error（logic 层通用入口）。
// code 传 pb 业务码（用 CodeNumXxx 常量）或 403/500，错误码体系见包注释。
func NewError(code int, msg string) error { return codes.NewCode(code, msg).Err() }

// Page 组装分页 data（admin handler 通用）。
func Page[T any](list []T, total int64, pageIndex, pageSize int) util.PageResp[T] {
	return util.PageResp[T]{List: list, Count: total, PageIndex: pageIndex, PageSize: pageSize}
}
