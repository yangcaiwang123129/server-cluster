package dto

// ClientVersionListReq 客户端版本分页列表请求。
type ClientVersionListReq struct {
	PageReq
	Version  string
	Platform string
	Status   string
	Order    string
}

// ClientVersionSaveReq 新增/修改客户端版本请求。
type ClientVersionSaveReq struct {
	VersionId   int    `json:"versionId"`
	Version     string `json:"version"`
	Platform    string `json:"platform"`
	DownloadURL string `json:"downloadUrl"`
	ForceUpdate bool   `json:"forceUpdate"`
	Status      string `json:"status"`
	ReleaseNote string `json:"releaseNote"`
}

// ClientVersionStatusReq 客户端版本启停请求。
type ClientVersionStatusReq struct {
	VersionId int    `json:"versionId"`
	Status    string `json:"status"`
}
