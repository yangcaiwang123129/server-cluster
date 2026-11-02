// Package logic 业务实现：实现 global 定义的接口；单例容器在本包，init 自注册（替代 app/wire）。
package logic

import (
	"server-cluster/node/battle/app/global"
	"server-cluster/share/singleton"
)

type battleLogic struct{}

var battleLogicInst = singleton.NewContainer[global.BattleLogic]("BattleLogic")

func init() {
	battleLogicInst.Register(&battleLogic{})
}

// Battle 获取战斗模块业务单例。
func Battle() global.BattleLogic { return battleLogicInst.MustGet() }
