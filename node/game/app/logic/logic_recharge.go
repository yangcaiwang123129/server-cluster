package logic

import (
	"context"

	"github.com/dobyte/due/v2/log"

	"server-cluster/node/game/app/global"
	"server-cluster/share/errs"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
	"server-cluster/share/tpl"
)

type rechargeLogic struct{}

var rechargeLogicInst = singleton.NewContainer[global.RechargeLogic]("RechargeLogic")

func init() {
	rechargeLogicInst.Register(&rechargeLogic{})
}

// Recharge 获取充值业务单例。
func Recharge() global.RechargeLogic { return rechargeLogicInst.MustGet() }

// notifyRechargeOrder 预留：通知后台充值订单。
// 接入支付 SDK 后，这里改为携带订单号回调支付平台/后台校验，校验通过再发放奖励；
// 当前是假充值，只记录一条日志，便于后续替换。
var notifyRechargeOrder = func(playerID int64, cfgID int32) {
	log.Infof("logic: recharge order notify reserved, playerID=%d, cfgID=%d", playerID, cfgID)
}

// Recharge 充值（假充值）：校验配置后直接发放 AssetReward，不做订单校验。
func (this *rechargeLogic) Recharge(ctx context.Context, playerID int64, req *pb.RechargeReq) (*pb.RechargeResp, error) {
	cfg := tpl.T().Recharge.Get(req.GetCfgId())
	if cfg == nil {
		return nil, errs.CodeInvalidParams.WithMessage("充值配置不存在").Err()
	}

	notifyRechargeOrder(playerID, req.GetCfgId())
	rewards, err := grantAssets(ctx, playerID, []*tpl.Asset{cfg.AssetReward})
	if err != nil {
		return nil, err
	}
	return &pb.RechargeResp{Rewards: rewards}, nil
}
