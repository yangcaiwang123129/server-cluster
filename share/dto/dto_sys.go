package dto

import "server-cluster/share/model"

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

// CaptchaResp 后台图形验证码响应。
type CaptchaResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Id   string `json:"id"`
	Data string `json:"data"`
}

// AdminLoginReq 后台登录请求：账号密码 + 图形验证码。
type AdminLoginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Code     string `json:"code"`
	UUID     string `json:"uuid"`
}

// GetInfoResp 后台当前登录用户的角色、权限与资料。
type GetInfoResp struct {
	Roles        []string `json:"roles"`
	Name         string   `json:"name"`
	Avatar       string   `json:"avatar"`
	Introduction string   `json:"introduction"`
	Permissions  []string `json:"permissions"`
}

// UserProfileResp 个人中心：当前用户资料与角色名称。
type UserProfileResp struct {
	User     *model.SysUser `json:"user"`
	RoleName string         `json:"roleName"`
}

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

// UserListReq 后台用户分页列表请求。
type UserListReq struct {
	PageReq
	Username string
	Phone    string
	Status   string
	Order    string
}

// UserSaveReq 新增/修改后台用户请求；Password 为空表示不修改。
type UserSaveReq struct {
	UserId   int    `json:"userId"`
	Username string `json:"username"`
	Password string `json:"password"`
	NickName string `json:"nickName"`
	Phone    string `json:"phone"`
	RoleId   int    `json:"roleId"`
	Avatar   string `json:"avatar"`
	Sex      string `json:"sex"`
	Email    string `json:"email"`
	Remark   string `json:"remark"`
	Status   string `json:"status"`
}

// UserStatusReq 启用/停用用户请求。
type UserStatusReq struct {
	UserId int    `json:"userId"`
	Status string `json:"status"`
}

// UserPasswordResetReq 管理员重置用户密码请求。
type UserPasswordResetReq struct {
	UserId   int    `json:"userId"`
	Password string `json:"password"`
}

// UserPasswordSetReq 当前用户修改密码请求。
type UserPasswordSetReq struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

// UserProfileUpdateReq 当前用户资料修改请求。
type UserProfileUpdateReq struct {
	NickName string `json:"nickName"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Sex      string `json:"sex"`
}
