package dto

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
