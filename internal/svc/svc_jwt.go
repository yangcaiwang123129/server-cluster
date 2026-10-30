package svc

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/dobyte/jwt"
)

// PayloadPlayerID JWT 载荷中的玩家 ID 键名；签发与验签必须使用同一键。
// 值以字符串写入：雪花 ID 超出 float64 精度（2^53），按数字存会丢低位。
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
// 兼容新格式（字符串）与旧 token（JSON 数字）。
func ParseToken(token string) (int64, error) {
	payload, err := JWT().ExtractPayload(token)
	if err != nil {
		return 0, err
	}
	v, ok := payload[PayloadPlayerID]
	if !ok {
		return 0, fmt.Errorf("missing %s in token", PayloadPlayerID)
	}
	return toInt64(v)
}

// toInt64 把载荷中的数值转换为 int64：字符串走精确解析，数字按 JSON 反序列化后的类型处理。
func toInt64(v any) (int64, error) {
	switch n := v.(type) {
	case string:
		id, err := strconv.ParseInt(n, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid %s value %q: %w", PayloadPlayerID, n, err)
		}
		return id, nil
	case float64:
		return int64(n), nil
	case json.Number:
		id, err := n.Int64()
		if err != nil {
			return 0, fmt.Errorf("invalid %s number %q: %w", PayloadPlayerID, n, err)
		}
		return id, nil
	default:
		return 0, fmt.Errorf("invalid %s type %T", PayloadPlayerID, v)
	}
}
