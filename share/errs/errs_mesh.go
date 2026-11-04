// Package errs 统一错误出口：admin HTTP 侧与 node proto 侧共用一套错误码。
//
// 错误码体系（pb common.proto ErrCode 为唯一定义点，客户端同用）：
//
//	0     成功（= pb.ErrCode_ERR_CODE_OK）
//	403   无接口访问权限（admin HTTP 语义补充，不进 pb）
//	500   内部错误（admin HTTP 语义补充，不进 pb）
//
// node proto 侧 Result.Code 直接用 pb.ErrCode（见 errs_node.go）；
// admin HTTP 侧 Resp.Code 用同一套 int 值（本文件）。
//
// 本文件所有函数参数统一为 fiber.Ctx：due 的 http.Context 内嵌 fiber.Ctx，
// handler 传 http.Context、middleware 传 fiber.Ctx 均可直接调用。
package errs

import (
	"errors"
	"server-cluster/share/pb"
	"strings"

	"server-cluster/share/dto"

	"github.com/dobyte/due/v2/codes"
	"github.com/dobyte/due/v2/log"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// Resp.code 统一取值（定义点在 pb common.proto ErrCode 枚举，这里只做 int 别名）。
const (
	// RespCodeOK 成功（= pb.ErrCode_ERR_CODE_OK）。
	RespCodeOK = int(pb.ErrCode_ERR_CODE_OK)
	// RespCodeForbidden 无接口访问权限（登录有效但 RBAC 拒绝，或账号/角色被停用）。
	RespCodeForbidden = int(pb.ErrCode_ERR_CODE_FORBIDDEN)
	// RespCodeInternal 内部错误兜底（未知错误、依赖故障、上游不可达）。
	RespCodeInternal = int(pb.ErrCode_ERR_CODE_INTERNAL)
)

// ============ 响应出口（唯一主体） ============

// Respond 统一响应出口。
//
//	err == nil                 → Resp{code:0, msg:"ok"[或 successMsg], data}
//	codes.Convert(err) != nil → Resp{code:业务码原样透传, msg}（0/1001/.../403/500）
//	gorm.ErrRecordNotFound    → Resp{code:500, msg:"记录不存在"}
//	Duplicate entry           → Resp{code:500, msg:"数据已存在"}
//	普通 error                → Resp{code:500, msg:"internal server error"} + log.Errorf
//
// 用法：
//
//	return errs.Respond(ctx, err, data)              // 成功 msg="ok"
//	return errs.Respond(ctx, nil, nil, "删除成功")    // 成功自定义 msg
//	return errs.Respond(ctx, errs.NewError(500, "x"), nil) // 手动码失败
//	return errs.Respond(ctx, err, nil)               // 透传 error 自动分类
func Respond(c fiber.Ctx, err error, data any, successMsg ...string) error {
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

// classify 错误分类：返回 (业务码, msg)。
func classify(err error) (int, string) {
	if c := codes.Convert(err); c != nil {
		return c.Code(), c.Message() // 业务码/403/500 原样透传（构造点保证合法性）
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

// ============ 便捷封装 ============

// OK 成功响应（data 传 nil + msg 即自定义消息）。
func OK(c fiber.Ctx, data any, successMsg ...string) error {
	return Respond(c, nil, data, successMsg...)
}

// Fail 失败响应，手动指定业务码（pb 业务码或 403/500）。
func Fail(c fiber.Ctx, code int, msg string) error {
	return Respond(c, codes.NewCode(code, msg).Err(), nil)
}

// FailErr 失败响应，透传 error 由 classify 自动分类。
func FailErr(c fiber.Ctx, err error) error { return Respond(c, err, nil) }

// NewError 创建带业务码的 error（logic 层通用；code 传 pb 业务码或 403/500）。
func NewError(code int, msg string) error { return codes.NewCode(code, msg).Err() }
