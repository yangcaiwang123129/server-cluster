package dto

import "encoding/json"

// SimOptionsResp 客户端模拟器默认参数。
type SimOptionsResp struct {
	LoginAddr string `json:"loginAddr"` // 登录服地址
}

// SimLoginReq 模拟客户端登录/注册请求。
type SimLoginReq struct {
	Mode      string `json:"mode"`      // login | register
	Username  string `json:"username"`  // 账号
	Password  string `json:"password"`  // 密码
	LoginAddr string `json:"loginAddr"` // 登录服地址，空则取默认配置
}

// SimLoginResp 登录/注册结果，JSON 中的 request/response 为页面展示用的报文。
type SimLoginResp struct {
	SessionID string          `json:"sessionId"`
	PlayerID  int64           `json:"playerId"`
	Token     string          `json:"token"`
	GateAddr  string          `json:"gateAddr"`
	TakenMs   int64           `json:"takenMs"`
	Request   json.RawMessage `json:"request"`
	Response  json.RawMessage `json:"response"`
}

// SimConnectResp 连接网关结果。
type SimConnectResp struct {
	GateAddr   string `json:"gateAddr"`
	RemoteAddr string `json:"remoteAddr"`
	LocalAddr  string `json:"localAddr"`
	TakenMs    int64  `json:"takenMs"`
}

// SimSessionReq 会话操作请求。
type SimSessionReq struct {
	SessionID string `json:"sessionId"`
}

// SimStepReq 流程步骤请求，step 取 verify | enter | leave。
type SimStepReq struct {
	SessionID string `json:"sessionId"`
	Step      string `json:"step"`
}

// SimStepResp 流程步骤结果，request/response 为收发报文的 JSON 表示。
type SimStepResp struct {
	Step     string          `json:"step"`
	Route    int32           `json:"route"`
	Seq      int32           `json:"seq"`
	Request  json.RawMessage `json:"request"`
	Response json.RawMessage `json:"response"`
	TakenMs  int64           `json:"takenMs"`
}
