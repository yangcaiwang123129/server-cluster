package errs

import (
	"server-cluster/share/pb"

	"github.com/dobyte/due/v2/codes"
)

// 业务错误码：数值与 common.proto ErrCode 枚举一致（pb 为唯一定义点，客户端同用）。
// 号段约定：0 成功；1xxx 通用；2xxx 账号；3xxx 玩家；4xxx 活动/任务。
//
// 用法（logic 层）：
//
//	return nil, errs.CodeInvalidParams.WithMessage("xx不能为空").Err()  // 透传 codes.Convert 能识别
//	return nil, codes.NewCode(400, "自定义错误").Err()                  // 也能识别
var (
	CodeOK              = codes.NewCode(int(pb.ErrCode_ERR_CODE_OK), "success")
	CodeInternalError   = codes.NewCode(int(pb.ErrCode_ERR_CODE_INTERNAL_ERROR), "internal error")
	CodeInvalidParams   = codes.NewCode(int(pb.ErrCode_ERR_CODE_INVALID_PARAMS), "invalid params")
	CodeUnauthorized    = codes.NewCode(int(pb.ErrCode_ERR_CODE_UNAUTHORIZED), "unauthorized")
	CodePlayerNotFound  = codes.NewCode(int(pb.ErrCode_ERR_CODE_PLAYER_NOT_FOUND), "player not found")
	CodePlayerExists    = codes.NewCode(int(pb.ErrCode_ERR_CODE_PLAYER_EXISTS), "player already exists")
	CodePasswordWrong   = codes.NewCode(int(pb.ErrCode_ERR_CODE_PASSWORD_WRONG), "password wrong")
	CodeTokenInvalid    = codes.NewCode(int(pb.ErrCode_ERR_CODE_TOKEN_INVALID), "token invalid")
	CodePlayerBanned    = codes.NewCode(int(pb.ErrCode_ERR_CODE_PLAYER_BANNED), "player is banned")
	CodeSmsCodeInvalid  = codes.NewCode(int(pb.ErrCode_ERR_CODE_SMS_CODE_INVALID), "sms code invalid or expired")
	CodeSmsCooldown     = codes.NewCode(int(pb.ErrCode_ERR_CODE_SMS_COOLDOWN), "please wait before requesting another sms code")
	CodeAuthTypeUnknown = codes.NewCode(int(pb.ErrCode_ERR_CODE_AUTH_TYPE_UNKNOWN), "unknown auth type")
)

// 活动/任务/活动快照错误码（数值与 common.proto ErrCode 一致）。
var (
	CodeActivityNotFound       = codes.NewCode(int(pb.ErrCode_ERR_CODE_ACTIVITY_NOT_FOUND), "activity not found")
	CodeActivityNotRunning     = codes.NewCode(int(pb.ErrCode_ERR_CODE_ACTIVITY_NOT_RUNNING), "activity not running")
	CodeActivityAlreadyClaimed = codes.NewCode(int(pb.ErrCode_ERR_CODE_ACTIVITY_ALREADY_CLAIMED), "activity reward already claimed")
	CodeSignInAlready          = codes.NewCode(int(pb.ErrCode_ERR_CODE_SIGN_IN_ALREADY), "already signed in today")
	CodeTaskNotFound           = codes.NewCode(int(pb.ErrCode_ERR_CODE_TASK_NOT_FOUND), "task not found")
	CodeTaskNotCompleted       = codes.NewCode(int(pb.ErrCode_ERR_CODE_TASK_NOT_COMPLETED), "task not completed")
	CodeTaskAlreadyClaimed     = codes.NewCode(int(pb.ErrCode_ERR_CODE_TASK_ALREADY_CLAIMED), "task reward already claimed")
	CodeTaskExpired            = codes.NewCode(int(pb.ErrCode_ERR_CODE_TASK_EXPIRED), "task period expired")
	CodeTaskNotAccepted        = codes.NewCode(int(pb.ErrCode_ERR_CODE_TASK_NOT_ACCEPTED), "task not accepted")
	CodeActivityUnavailable    = codes.NewCode(int(pb.ErrCode_ERR_CODE_ACTIVITY_UNAVAILABLE), "activity snapshot unavailable")
)

// BuildOKResult 构造成功 Result（Code = ERR_CODE_OK）。
func BuildOKResult() *pb.Result {
	return &pb.Result{Code: pb.ErrCode_ERR_CODE_OK}
}

// BuildErrResult 手动构造业务错误 Result（logic/handler 层通用）。
//
//	code: pb 业务码枚举值
//	msg:  自定义消息（传 "" 则客户端使用 code 对应默认文案）
//	args: 可变参数（可选，透传给客户端做参数替换）
func BuildErrResult(code pb.ErrCode, msg string, args ...string) *pb.Result {
	return &pb.Result{
		Code: code,
		Msg:  msg,
		Args: args,
	}
}
