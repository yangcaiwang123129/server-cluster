package consts

import (
	"github.com/dobyte/due/v2/codes"

	"server-cluster/share/pb"
)

// 业务错误码：数值与 common.proto ErrCode 枚举一致（pb 为唯一定义点，客户端同用）。
// 号段约定：0 成功；1xxx 通用；2xxx 账号；3xxx 玩家。
var (
	CodeOK              = codes.NewCode(int(pb.ErrCode_ERR_CODE_OK), "success")
	CodeInternalError   = codes.NewCode(int(pb.ErrCode_ERR_CODE_INTERNAL_ERROR), "internal error")
	CodeInvalidParams   = codes.NewCode(int(pb.ErrCode_ERR_CODE_INVALID_PARAMS), "invalid params")
	CodeUnauthorized    = codes.NewCode(int(pb.ErrCode_ERR_CODE_UNAUTHORIZED), "unauthorized")
	CodeAccountNotFound = codes.NewCode(int(pb.ErrCode_ERR_CODE_ACCOUNT_NOT_FOUND), "account not found")
	CodeAccountExists   = codes.NewCode(int(pb.ErrCode_ERR_CODE_ACCOUNT_EXISTS), "account already exists")
	CodePasswordWrong   = codes.NewCode(int(pb.ErrCode_ERR_CODE_PASSWORD_WRONG), "password wrong")
	CodeTokenInvalid    = codes.NewCode(int(pb.ErrCode_ERR_CODE_TOKEN_INVALID), "token invalid")
	CodePlayerNotFound  = codes.NewCode(int(pb.ErrCode_ERR_CODE_PLAYER_NOT_FOUND), "player not found")
)
