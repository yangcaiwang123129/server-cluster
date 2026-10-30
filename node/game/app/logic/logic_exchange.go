package logic

import (
	"context"

	"server-cluster/node/game/app/global"
	"server-cluster/share/errs"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
	"server-cluster/share/tpl"
)

type exchangeLogic struct{}

var exchangeLogicInst = singleton.NewContainer[global.ExchangeLogic]("ExchangeLogic")

func init() {
	exchangeLogicInst.Register(&exchangeLogic{})
}

// Exchange 获取兑换业务单例。
func Exchange() global.ExchangeLogic { return exchangeLogicInst.MustGet() }

// Exchange 兑换：按 ExchangeCfg 扣除 AssetCost 并发放 AssetRewards。
// 配置不存在或资产不足返回错误，已扣减的部分不回滚（与现有资产逻辑一致）。
func (this *exchangeLogic) Exchange(ctx context.Context, playerID int64, req *pb.ExchangeReq) (*pb.ExchangeResp, error) {
	cfg := tpl.T().Exchange.Get(req.GetCfgId())
	if cfg == nil {
		return nil, errs.CodeInvalidParams.WithMessage("兑换配置不存在").Err()
	}

	costs, err := deductAssets(ctx, playerID, []*tpl.Asset{cfg.AssetCost})
	if err != nil {
		return nil, err
	}
	rewards, err := grantAssets(ctx, playerID, cfg.AssetRewards)
	if err != nil {
		return nil, err
	}
	return &pb.ExchangeResp{Costs: costs, Rewards: rewards}, nil
}
