package svc

import (
	"fmt"

	"github.com/dobyte/jwt"
)

// PayloadPlayerID JWT 载荷中的玩家 ID 键名；签发与验签必须使用同一键。
const PayloadPlayerID = "player_id"

// JWT 获取令牌签发/校验器；未配置 etc.jwt.secret 时 panic。
func JWT() *jwt.JWT {
	j := Ctx().JWT
	if j == nil {
		panic("svc jwt is not initialized, check etc.jwt.secret")
	}
	return j
}

// ParseToken 校验 login 签发的 JWT 并提取 player_id。
func ParseToken(token string) (int64, error) {
	payload, err := JWT().ExtractPayload(token)
	if err != nil {
		return 0, err
	}
	v, ok := payload[PayloadPlayerID]
	if !ok {
		return 0, fmt.Errorf("missing %s in token", PayloadPlayerID)
	}
	f, ok := v.(float64)
	if !ok {
		return 0, fmt.Errorf("invalid %s type in token", PayloadPlayerID)
	}
	return int64(f), nil
}
