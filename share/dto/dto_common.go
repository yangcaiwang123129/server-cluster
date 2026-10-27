package dto

// 管理后台公共出入参：只放数据结构，不放业务逻辑。

// Resp 统一响应信封：前端按 code 判断成败。
type Resp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data,omitempty"`
}

// TokenResp 后台登录响应：token 与信封同级。
type TokenResp struct {
	Code  int    `json:"code"`
	Msg   string `json:"msg"`
	Token string `json:"token"`
}

// PageReq 分页参数。
type PageReq struct {
	PageIndex int
	PageSize  int
}

// IdsReq 批量操作请求。
type IdsReq struct {
	Ids []int `json:"ids"`
}

// Option 前端下拉项。
type Option struct {
	Label string `json:"label"`
	Value string `json:"value"`
}
