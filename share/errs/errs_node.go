package errs

import (
	"server-cluster/share/pb"

	"github.com/dobyte/due/v2/codes"
)

// 业务错误码：定义点在 common.proto ErrCode 枚举（pb 为唯一定义点，客户端同用）。
// 号段约定：0 成功；403/500 admin 补充；1xxx 通用；2xxx 账号；3xxx 登录鉴权；4xxx 活动/任务。
//
// 用法（logic 层）：
//
//	return nil, errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "xx不能为空") // codes.Convert 能识别
//	return &pb.XxxResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_XXX, "msg")}  // 校验失败直接填 Result
func ErrOf(code pb.ErrCode, msg string) error {
	return codes.NewCode(int(code), msg).Err()
}

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
