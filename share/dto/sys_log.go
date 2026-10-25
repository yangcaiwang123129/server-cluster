package dto

// LoginLogListReq 后台登录日志分页列表请求。
type LoginLogListReq struct {
	PageReq
	Username  string
	Ipaddr    string
	Status    string
	BeginTime string
	EndTime   string
	Order     string
}

// OperaLogListReq 后台操作日志分页列表请求。
type OperaLogListReq struct {
	PageReq
	Title     string
	OperName  string
	Status    string
	BeginTime string
	EndTime   string
	Order     string
}
