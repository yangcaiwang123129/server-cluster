package dto

// ApiListReq 后台接口登记分页列表请求。
type ApiListReq struct {
	PageReq
	Title  string
	Path   string
	Action string
	Type   string
	Order  string
}

// ApiSaveReq 新增/修改接口请求。
type ApiSaveReq struct {
	Id     int    `json:"id"`
	Path   string `json:"path"`
	Handle string `json:"handle"`
	Title  string `json:"title"`
	Type   string `json:"type"`
	Action string `json:"action"`
}
