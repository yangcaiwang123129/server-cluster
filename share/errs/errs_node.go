package errs

import (
	"server-cluster/share/pb"

	"github.com/dobyte/due/v2/codes"
)

// ErrOf 构造 logic 层业务错误（codes.Convert 可识别），号段约定见 pb common.proto ErrCode。
func ErrOf(code pb.ErrCode, msg string) error {
	return codes.NewCode(int(code), msg).Err()
}

// BuildOKResult 构造成功 Result（Code = ERR_CODE_OK）。
func BuildOKResult() *pb.Result {
	return &pb.Result{Code: pb.ErrCode_ERR_CODE_OK}
}

// BuildErrResult 手动构造业务错误 Result（logic/handler 层通用，校验失败直接填 Resp.Result）。
func BuildErrResult(code pb.ErrCode, msg string, args ...string) *pb.Result {
	return &pb.Result{
		Code: code,
		Msg:  msg,
		Args: args,
	}
}
