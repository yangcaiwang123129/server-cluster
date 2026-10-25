package dto

// DictTypeListReq 字典类型分页列表请求。
type DictTypeListReq struct {
	PageReq
	DictName string
	DictType string
	Status   string
	Order    string
}

// DictTypeSaveReq 新增/修改字典类型请求；status 前端按数字传输。
type DictTypeSaveReq struct {
	Id       int    `json:"id"`
	DictName string `json:"dictName"`
	DictType string `json:"dictType"`
	Status   int    `json:"status"`
	Remark   string `json:"remark"`
}

// DictDataListReq 字典项分页列表请求。
type DictDataListReq struct {
	PageReq
	DictType  string
	DictLabel string
	Status    string
	Order     string
}

// DictDataSaveReq 新增/修改字典项请求；status 前端按数字传输。
type DictDataSaveReq struct {
	DictCode  int    `json:"dictCode"`
	DictSort  int    `json:"dictSort"`
	DictLabel string `json:"dictLabel"`
	DictValue string `json:"dictValue"`
	DictType  string `json:"dictType"`
	CssClass  string `json:"cssClass"`
	ListClass string `json:"listClass"`
	IsDefault string `json:"isDefault"`
	Status    int    `json:"status"`
	Remark    string `json:"remark"`
}
