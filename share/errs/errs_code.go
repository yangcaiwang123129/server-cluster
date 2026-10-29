package errs

import (
	"server-cluster/share/pb"

	"github.com/dobyte/due/v2/codes"
)

// 业务错误码：数值与 common.proto ErrCode 枚举一致（pb 为唯一定义点，客户端同用）。
// 号段约定：0 成功；1xxx 通用；2xxx 账号；3xxx 玩家。
//
// 用法（logic 层）：
//
//	return nil, errs.CodeInvalidParams.WithMessage("xx不能为空").Err()  // 透传 codes.Convert 能识别
//	return nil, codes.NewCode(400, "自定义错误").Err()                  // 也能识别
var (
	CodeOK             = codes.NewCode(int(pb.ErrCode_ERR_CODE_OK), "success")
	CodeInternalError  = codes.NewCode(int(pb.ErrCode_ERR_CODE_INTERNAL_ERROR), "internal error")
	CodeInvalidParams  = codes.NewCode(int(pb.ErrCode_ERR_CODE_INVALID_PARAMS), "invalid params")
	CodeUnauthorized   = codes.NewCode(int(pb.ErrCode_ERR_CODE_UNAUTHORIZED), "unauthorized")
	CodePlayerNotFound = codes.NewCode(int(pb.ErrCode_ERR_CODE_PLAYER_NOT_FOUND), "player not found")
	CodePlayerExists   = codes.NewCode(int(pb.ErrCode_ERR_CODE_PLAYER_EXISTS), "player already exists")
	CodePasswordWrong  = codes.NewCode(int(pb.ErrCode_ERR_CODE_PASSWORD_WRONG), "password wrong")
	CodeTokenInvalid   = codes.NewCode(int(pb.ErrCode_ERR_CODE_TOKEN_INVALID), "token invalid")
)

// ErrXxx 便捷入口：直接拿到 error（CodeXxx.Err()）。
// 优先级：logic 层推荐 CodeXxx.WithMessage("xxx").Err() 链式覆盖；旧代码可用 ErrXxx。
var (
	ErrOK             = CodeOK.Err()
	ErrInternalError  = CodeInternalError.Err()
	ErrInvalidParams  = CodeInvalidParams.Err()
	ErrUnauthorized   = CodeUnauthorized.Err()
	ErrPlayerNotFound = CodePlayerNotFound.Err()
	ErrPlayerExists   = CodePlayerExists.Err()
	ErrPasswordWrong  = CodePasswordWrong.Err()
	ErrTokenInvalid   = CodeTokenInvalid.Err()
)
