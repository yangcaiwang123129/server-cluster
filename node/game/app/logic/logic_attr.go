package logic

import (
	"context"
	"encoding/json"
	"strings"
	"unicode/utf8"

	"server-cluster/node/game/app/global"
	"server-cluster/share/consts"
	"server-cluster/share/errs"
	"server-cluster/share/model"
	"server-cluster/share/pb"
)

// 属性更新实现：校验并写入 core，返回操作后的属性值。
type attrUpdateHandler func(core *model.PlayerExtra, value *pb.ObjectValue) (*pb.ObjectValue, error)

// 属性领取实现：按玩家状态发放奖励，返回操作后的属性值。
type attrClaimHandler func(ctx context.Context, playerID int64, core *model.PlayerExtra) (*pb.ObjectValue, error)

// attrHandlers 属性类型 → 操作实现；未登记的类型或未实现的操作一律视为不支持。
// 1000~4999 任务/成就进度、5000~9999 活动状态等属性接入时在此扩展。
var attrHandlers = map[pb.Attr]struct {
	update attrUpdateHandler
	claim  attrClaimHandler
}{
	pb.Attr_PLAYER_ATTR_NAME:          {update: updateNickname},
	pb.Attr_PLAYER_ATTR_SIGNATURE:     {update: updateSignature},
	pb.Attr_PLAYER_ATTR_SETTINGS_JSON: {update: updateSettingsJSON},
}

// AttrOperate 属性接口：客户端传属性类型与操作，按属性类型分发处理。
func (this *playerLogic) AttrOperate(ctx context.Context, playerID int64, req *pb.AttrOperateReq) (*pb.AttrOperateResp, error) {
	attr := pb.Attr(req.GetAttr())
	h, ok := attrHandlers[attr]
	if !ok {
		return &pb.AttrOperateResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "不支持的属性类型")}, nil
	}

	core, err := global.PlayerExtraCache.Get(ctx, playerID)
	if err != nil {
		return nil, err
	}

	var out *pb.ObjectValue
	switch req.GetOperate() {
	case pb.AttrOperate_UPDATE:
		if h.update == nil {
			return &pb.AttrOperateResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "该属性不支持更新")}, nil
		}
		out, err = h.update(core, req.GetValue())
	case pb.AttrOperate_CLAIM:
		if h.claim == nil {
			return &pb.AttrOperateResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "该属性不支持领取")}, nil
		}
		out, err = h.claim(ctx, playerID, core)
	default:
		return &pb.AttrOperateResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "不支持的操作类型")}, nil
	}
	if err != nil {
		if r := codedResult(err); r != nil {
			return &pb.AttrOperateResp{Result: r}, nil
		}
		return nil, err
	}

	global.PlayerExtraCache.Set(playerID, core)
	return &pb.AttrOperateResp{Values: map[int32]*pb.ObjectValue{int32(attr): out}}, nil
}

// updateNickname 更新昵称：去首尾空白，长度 1~MaxNicknameLen 字符。
func updateNickname(core *model.PlayerExtra, value *pb.ObjectValue) (*pb.ObjectValue, error) {
	name := strings.TrimSpace(value.GetStringValue())
	n := utf8.RuneCountInString(name)
	if n == 0 || n > consts.MaxNicknameLen {
		return nil, errs.CodeInvalidParams.WithMessage("昵称长度不合法").Err()
	}
	core.Nickname = name
	return &pb.ObjectValue{Kind: &pb.ObjectValue_StringValue{StringValue: core.Nickname}}, nil
}

// updateSignature 更新个性签名：长度不超过 MaxSignatureLen 字符，允许为空。
func updateSignature(core *model.PlayerExtra, value *pb.ObjectValue) (*pb.ObjectValue, error) {
	signature := strings.TrimSpace(value.GetStringValue())
	if utf8.RuneCountInString(signature) > consts.MaxSignatureLen {
		return nil, errs.CodeInvalidParams.WithMessage("个性签名过长").Err()
	}
	core.Signature = signature
	return &pb.ObjectValue{Kind: &pb.ObjectValue_StringValue{StringValue: core.Signature}}, nil
}

// updateSettingsJSON 更新个人设置：必须是 JSON 对象且不超过 MaxSettingsJSONLen 字节。
func updateSettingsJSON(core *model.PlayerExtra, value *pb.ObjectValue) (*pb.ObjectValue, error) {
	raw := strings.TrimSpace(value.GetStringValue())
	if raw == "" || len(raw) > consts.MaxSettingsJSONLen {
		return nil, errs.CodeInvalidParams.WithMessage("个人设置长度不合法").Err()
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &obj); err != nil || obj == nil {
		return nil, errs.CodeInvalidParams.WithMessage("个人设置必须是 JSON 对象").Err()
	}
	core.SettingsJSON = raw
	return &pb.ObjectValue{Kind: &pb.ObjectValue_StringValue{StringValue: core.SettingsJSON}}, nil
}
