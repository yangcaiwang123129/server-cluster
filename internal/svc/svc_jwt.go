package svc

import (
	"fmt"

	"github.com/dobyte/jwt"
)

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
	v, ok := payload["player_id"]
	if !ok {
		return 0, fmt.Errorf("missing player_id in token")
	}
	f, ok := v.(float64)
	if !ok {
		return 0, fmt.Errorf("invalid player_id type in token")
	}
	return int64(f), nil
}
