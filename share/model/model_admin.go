package model

import (
	"time"

	"server-cluster/share/enum"
)

// 管理后台（mesh/admin）数据表模型。
// 与前端 mesh/admin-ui 的字段契约一一对应，由 gorm AutoMigrate 维护；不含多租户字段。

// SysUser 后台运营账号。
type SysUser struct {
	UserId    int       `json:"userId" gorm:"column:user_id;primaryKey;autoIncrement"`
	Username  string    `json:"username" gorm:"column:username;size:64;not null;uniqueIndex"`
	Password  string    `json:"-" gorm:"column:password;size:128;not null"`
	NickName  string    `json:"nickName" gorm:"column:nick_name;size:64"`
	Phone     string    `json:"phone" gorm:"column:phone;size:32"`
	RoleId    int       `json:"roleId" gorm:"column:role_id;index"`
	Avatar    string    `json:"avatar" gorm:"column:avatar;size:255"`
	Sex       string    `json:"sex" gorm:"column:sex;size:8"`
	Email     string    `json:"email" gorm:"column:email;size:128"`
	Remark    string    `json:"remark" gorm:"column:remark;size:255"`
	Status    string    `json:"status" gorm:"column:status;size:4;default:2"` // 1 停用 2 正常
	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"column:updated_at"`
}

// TableName 表名。
func (SysUser) TableName() string { return enum.TableSysUser.String() }

// SysRole 角色。
type SysRole struct {
	RoleId    int       `json:"roleId" gorm:"column:role_id;primaryKey;autoIncrement"`
	RoleName  string    `json:"roleName" gorm:"column:role_name;size:64;not null"`
	RoleKey   string    `json:"roleKey" gorm:"column:role_key;size:64;not null;uniqueIndex"`
	RoleSort  int       `json:"roleSort" gorm:"column:role_sort"`
	Status    string    `json:"status" gorm:"column:status;size:4;default:2"` // 1 停用 2 正常
	Remark    string    `json:"remark" gorm:"column:remark;size:255"`
	MenuIds   []int     `json:"menuIds" gorm:"-"`
	Menus     []SysMenu `json:"menus,omitempty" gorm:"many2many:sys_role_menu;joinForeignKey:RoleId;joinReferences:MenuId"`
	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"column:updated_at"`
}

// TableName 表名。
func (SysRole) TableName() string { return enum.TableSysRole.String() }

// SysMenu 菜单/按钮权限。菜单树是前端动态路由的唯一来源。
type SysMenu struct {
	MenuId     int       `json:"menuId" gorm:"column:menu_id;primaryKey;autoIncrement"`
	MenuName   string    `json:"menuName" gorm:"column:menu_name;size:128"` // 前端路由 name
	Title      string    `json:"title" gorm:"column:title;size:128"`        // 显示标题
	Icon       string    `json:"icon" gorm:"column:icon;size:128"`
	Path       string    `json:"path" gorm:"column:path;size:128"`
	MenuType   string    `json:"menuType" gorm:"column:menu_type;size:1"` // M 目录 C 页面 F 按钮
	Action     string    `json:"action" gorm:"column:action;size:16"`
	Permission string    `json:"permission" gorm:"column:permission;size:255"` // 权限标识 模块:资源:操作
	ParentId   int       `json:"parentId" gorm:"column:parent_id;index"`
	NoCache    bool      `json:"noCache" gorm:"column:no_cache"`
	Component  string    `json:"component" gorm:"column:component;size:255"`
	Sort       int       `json:"sort" gorm:"column:sort"`
	Visible    string    `json:"visible" gorm:"column:visible;size:1;default:0"`  // 0 显示 1 隐藏
	IsFrame    string    `json:"isFrame" gorm:"column:is_frame;size:1;default:1"` // 1 否 0 是
	SysApi     []SysApi  `json:"sysApi,omitempty" gorm:"many2many:sys_menu_api_rule;joinForeignKey:MenuId;joinReferences:ApiId"`
	Apis       []int     `json:"apis" gorm:"-"`
	Children   []SysMenu `json:"children,omitempty" gorm:"-"`
	CreatedAt  time.Time `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt  time.Time `json:"updatedAt" gorm:"column:updated_at"`
}

// TableName 表名。
func (SysMenu) TableName() string { return enum.TableSysMenu.String() }

// SysApi 后台接口登记表：菜单勾选接口后，角色据此获得放行规则。
type SysApi struct {
	ApiId     int       `json:"id" gorm:"column:api_id;primaryKey;autoIncrement"`
	Path      string    `json:"path" gorm:"column:path;size:255;index"`
	Handle    string    `json:"handle" gorm:"column:handle;size:255"`
	Title     string    `json:"title" gorm:"column:title;size:255"`
	Type      string    `json:"type" gorm:"column:type;size:8;default:BUS"` // SYS 基础设施 BUS 业务
	Action    string    `json:"action" gorm:"column:action;size:16"`
	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"column:updated_at"`
}

// TableName 表名。
func (SysApi) TableName() string { return enum.TableSysApi.String() }

// SysDictType 字典类型。
type SysDictType struct {
	DictId    int       `json:"id" gorm:"column:dict_id;primaryKey;autoIncrement"`
	DictName  string    `json:"dictName" gorm:"column:dict_name;size:128;not null"`
	DictType  string    `json:"dictType" gorm:"column:dict_type;size:128;not null;uniqueIndex"`
	Status    string    `json:"status" gorm:"column:status;size:4;default:2"`
	Remark    string    `json:"remark" gorm:"column:remark;size:255"`
	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"column:updated_at"`
}

// TableName 表名。
func (SysDictType) TableName() string { return enum.TableSysDictType.String() }

// SysDictData 字典项。
type SysDictData struct {
	DictCode  int       `json:"dictCode" gorm:"column:dict_code;primaryKey;autoIncrement"`
	DictSort  int       `json:"dictSort" gorm:"column:dict_sort"`
	DictLabel string    `json:"dictLabel" gorm:"column:dict_label;size:128"`
	DictValue string    `json:"dictValue" gorm:"column:dict_value;size:128"`
	DictType  string    `json:"dictType" gorm:"column:dict_type;size:128;index"`
	CssClass  string    `json:"cssClass" gorm:"column:css_class;size:128"`
	ListClass string    `json:"listClass" gorm:"column:list_class;size:128"`
	IsDefault string    `json:"isDefault" gorm:"column:is_default;size:8;default:N"`
	Status    string    `json:"status" gorm:"column:status;size:4;default:2"`
	Remark    string    `json:"remark" gorm:"column:remark;size:255"`
	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"column:updated_at"`
}

// TableName 表名。
func (SysDictData) TableName() string { return enum.TableSysDictData.String() }

// SysConfig 系统配置项。
type SysConfig struct {
	ConfigId    int       `json:"id" gorm:"column:config_id;primaryKey;autoIncrement"`
	ConfigName  string    `json:"configName" gorm:"column:config_name;size:128"`
	ConfigKey   string    `json:"configKey" gorm:"column:config_key;size:128;not null;uniqueIndex"`
	ConfigValue string    `json:"configValue" gorm:"column:config_value;size:1024"`
	ConfigType  string    `json:"configType" gorm:"column:config_type;size:8;default:N"` // Y 内置 N 自定义
	IsFrontend  string    `json:"isFrontend" gorm:"column:is_frontend;size:8;default:0"`
	Remark      string    `json:"remark" gorm:"column:remark;size:255"`
	CreatedAt   time.Time `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt   time.Time `json:"updatedAt" gorm:"column:updated_at"`
}

// TableName 表名。
func (SysConfig) TableName() string { return enum.TableSysConfig.String() }

// SysLoginLog 登录审计。
type SysLoginLog struct {
	Id            int       `json:"id" gorm:"column:info_id;primaryKey;autoIncrement"`
	Username      string    `json:"username" gorm:"column:username;size:64;index"`
	Ipaddr        string    `json:"ipaddr" gorm:"column:ipaddr;size:64"`
	LoginLocation string    `json:"loginLocation" gorm:"column:login_location;size:128"`
	Browser       string    `json:"browser" gorm:"column:browser;size:128"`
	Os            string    `json:"os" gorm:"column:os;size:128"`
	Platform      string    `json:"platform" gorm:"column:platform;size:128"`
	Status        string    `json:"status" gorm:"column:status;size:4"` // 1 失败 2 成功
	Msg           string    `json:"msg" gorm:"column:msg;size:255"`
	LoginTime     time.Time `json:"loginTime" gorm:"column:login_time"`
	CreatedAt     time.Time `json:"createdAt" gorm:"column:created_at"`
}

// TableName 表名。
func (SysLoginLog) TableName() string { return enum.TableSysLoginLog.String() }

// SysOperaLog 操作审计。
type SysOperaLog struct {
	Id            int       `json:"id" gorm:"column:oper_id;primaryKey;autoIncrement"`
	Title         string    `json:"title" gorm:"column:title;size:128"`
	BusinessType  string    `json:"businessType" gorm:"column:business_type;size:8"`
	Method        string    `json:"method" gorm:"column:method;size:255"`
	RequestMethod string    `json:"requestMethod" gorm:"column:request_method;size:16"`
	OperUrl       string    `json:"operUrl" gorm:"column:oper_url;size:255"`
	OperIp        string    `json:"operIp" gorm:"column:oper_ip;size:64"`
	OperName      string    `json:"operName" gorm:"column:oper_name;size:64"`
	Status        string    `json:"status" gorm:"column:status;size:4"` // 1 失败 2 成功
	OperTime      time.Time `json:"operTime" gorm:"column:oper_time"`
	OperLocation  string    `json:"operLocation" gorm:"column:oper_location;size:128"`
	OperParam     string    `json:"operParam" gorm:"column:oper_param;size:2048"`
	JsonResult    string    `json:"jsonResult" gorm:"column:json_result;size:2048"`
	LatencyTime   string    `json:"latencyTime" gorm:"column:latency_time;size:16"`
	CreatedAt     time.Time `json:"createdAt" gorm:"column:created_at"`
}

// TableName 表名。
func (SysOperaLog) TableName() string { return enum.TableSysOperaLog.String() }

// CasbinRule 放行规则：ptype=p 时 v0=roleKey v1=path v2=method。
type CasbinRule struct {
	Id    int    `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Ptype string `json:"ptype" gorm:"column:ptype;size:16"`
	V0    string `json:"v0" gorm:"column:v0;size:128"`
	V1    string `json:"v1" gorm:"column:v1;size:255"`
	V2    string `json:"v2" gorm:"column:v2;size:16"`
	V3    string `json:"v3" gorm:"column:v3;size:255"`
	V4    string `json:"v4" gorm:"column:v4;size:255"`
	V5    string `json:"v5" gorm:"column:v5;size:255"`
}

// TableName 表名。
func (CasbinRule) TableName() string { return enum.TableCasbinRule.String() }
