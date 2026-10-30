package consts

import "time"

// 战斗房间参数（internal/combat）。
const (
	// CombatInitialHP 玩家入房时的初始生命值。
	CombatInitialHP = 100
	// CombatInitialAttack 玩家入房时的初始攻击。
	CombatInitialAttack = 100
	// CombatInitialDefense 玩家入房时的初始防御。
	CombatInitialDefense = 0
	// CombatInitialSpeed 玩家入房时的初始速度。
	CombatInitialSpeed = 100

	// CombatTickRate 房间逻辑帧间隔。
	CombatTickRate = 50 * time.Millisecond
	// CombatQueueSize 房间 actor 的消息队列长度。
	CombatQueueSize = 512
)
