package dto

// ConfigListReq 系统配置分页列表请求。
type ConfigListReq struct {
	PageReq
	ConfigName string
	ConfigKey  string
	ConfigType string
	Order      string
}

// ConfigSaveReq 新增/修改配置请求。
type ConfigSaveReq struct {
	Id          int    `json:"id"`
	ConfigName  string `json:"configName"`
	ConfigKey   string `json:"configKey"`
	ConfigValue string `json:"configValue"`
	ConfigType  string `json:"configType"`
	IsFrontend  string `json:"isFrontend"`
	Remark      string `json:"remark"`
}

// ConfigValueResp 按配置键取值响应。
type ConfigValueResp struct {
	ConfigValue string `json:"configValue"`
}

// SetConfigReq 系统配置页批量保存请求。
type SetConfigReq []SetConfigItem

// SetConfigItem 单个配置项。
type SetConfigItem struct {
	ConfigKey   string `json:"configKey"`
	ConfigValue string `json:"configValue"`
}
