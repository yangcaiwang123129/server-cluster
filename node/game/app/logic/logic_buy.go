package logic

import (
	"context"

	"server-cluster/node/game/app/global"
	"server-cluster/share/errs"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
	"server-cluster/share/tpl"
)

type buyLogic struct{}

var buyLogicInst = singleton.NewContainer[global.BuyLogic]("BuyLogic")

func init() {
	buyLogicInst.Register(&buyLogic{})
}

// Buy 获取礼包购买业务单例。
func Buy() global.BuyLogic { return buyLogicInst.MustGet() }

// GiftBuy 购买礼包：按 GiftCfg 扣除 AssetCost 并发放 AssetRewards。
// 配置不存在或资产不足返回错误，已扣减的部分不回滚（与现有资产逻辑一致）。
func (this *buyLogic) GiftBuy(ctx context.Context, playerID int64, req *pb.GiftBuyReq) (*pb.GiftBuyResp, error) {
	cfg := tpl.T().Gift.Get(req.GetCfgId())
	if cfg == nil {
		return nil, errs.CodeInvalidParams.WithMessage("礼包配置不存在").Err()
	}

	costs, err := deductAssets(ctx, playerID, []*tpl.Asset{cfg.AssetCost})
	if err != nil {
		return nil, err
	}
	rewards, err := grantAssets(ctx, playerID, cfg.AssetRewards)
	if err != nil {
		return nil, err
	}
	return &pb.GiftBuyResp{Costs: costs, Rewards: rewards}, nil
}
