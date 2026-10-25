package dto

import "server-cluster/share/model"

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
