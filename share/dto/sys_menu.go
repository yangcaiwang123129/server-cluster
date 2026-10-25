package dto

// MenuListReq 后台菜单树请求。
type MenuListReq struct {
	Title    string
	MenuName string
	Visible  string
}

// MenuSaveReq 新增/修改菜单请求。
type MenuSaveReq struct {
	MenuId     int    `json:"menuId"`
	MenuName   string `json:"menuName"`
	Title      string `json:"title"`
	Icon       string `json:"icon"`
	Path       string `json:"path"`
	MenuType   string `json:"menuType"`
	Action     string `json:"action"`
	Permission string `json:"permission"`
	ParentId   int    `json:"parentId"`
	NoCache    bool   `json:"noCache"`
	Component  string `json:"component"`
	Sort       int    `json:"sort"`
	Visible    string `json:"visible"`
	IsFrame    string `json:"isFrame"`
	Apis       []int  `json:"apis"`
}

// MenuNode 角色授权树节点。
type MenuNode struct {
	Id       int        `json:"id"`
	Label    string     `json:"label"`
	Children []MenuNode `json:"children,omitempty"`
}

// RoleMenuTreeselectResp 角色授权树响应。
type RoleMenuTreeselectResp struct {
	Menus       []MenuNode `json:"menus"`
	CheckedKeys []int      `json:"checkedKeys"`
}
