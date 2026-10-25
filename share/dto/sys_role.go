package dto

// RoleListReq 后台角色分页列表请求。
type RoleListReq struct {
	PageReq
	RoleName string
	RoleKey  string
	Status   string
	Order    string
}

// RoleSaveReq 新增/修改角色请求。
type RoleSaveReq struct {
	RoleId   int    `json:"roleId"`
	RoleName string `json:"roleName"`
	RoleKey  string `json:"roleKey"`
	RoleSort int    `json:"roleSort"`
	Status   string `json:"status"`
	Remark   string `json:"remark"`
	MenuIds  []int  `json:"menuIds"`
}

// RoleStatusReq 启用/停用角色请求。
type RoleStatusReq struct {
	RoleId int    `json:"roleId"`
	Status string `json:"status"`
}
