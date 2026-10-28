package model

import (
	"time"

	"server-cluster/share/enum"
)

// 管理后台（mesh/admin）数据表模型。
// 与前端 mesh/admin-ui 的字段契约一一对应，由 gorm AutoMigrate 维护；不含多租户字段。

// SysUser 后台运营账号。
type SysUser struct {
	UserId    int       `json:"userId" gorm:"column:user_id;type:int;not null;primaryKey;autoIncrement;comment:用户ID"`
	Username  string    `json:"username" gorm:"column:username;type:varchar(64);not null;default:'';uniqueIndex;comment:登录账号"`
	Password  string    `json:"-" gorm:"column:password;type:varchar(128);not null;default:'';comment:密码哈希"`
	NickName  string    `json:"nickName" gorm:"column:nick_name;type:varchar(64);not null;default:'';comment:昵称"`
	Phone     string    `json:"phone" gorm:"column:phone;type:varchar(32);not null;default:'';comment:手机号"`
	RoleId    int       `json:"roleId" gorm:"column:role_id;type:int;not null;default:0;index;comment:角色ID"`
	Avatar    string    `json:"avatar" gorm:"column:avatar;type:varchar(255);not null;default:'';comment:头像地址"`
	Sex       string    `json:"sex" gorm:"column:sex;type:varchar(8);not null;default:'';comment:性别(0未知 1男 2女)"`
	Email     string    `json:"email" gorm:"column:email;type:varchar(128);not null;default:'';comment:邮箱"`
	Remark    string    `json:"remark" gorm:"column:remark;type:varchar(255);not null;default:'';comment:备注"`
	Status    string    `json:"status" gorm:"column:status;type:varchar(4);not null;default:2;comment:状态(1停用 2正常)"`
	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at;type:datetime;not null;comment:创建时间"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"column:updated_at;type:datetime;not null;comment:更新时间"`
}

// TableName 表名。
func (SysUser) TableName() string { return enum.TableSysUser.String() }

// SysRole 角色。
type SysRole struct {
	RoleId    int       `json:"roleId" gorm:"column:role_id;type:int;not null;primaryKey;autoIncrement;comment:角色ID"`
	RoleName  string    `json:"roleName" gorm:"column:role_name;type:varchar(64);not null;default:'';comment:角色名称"`
	RoleKey   string    `json:"roleKey" gorm:"column:role_key;type:varchar(64);not null;default:'';uniqueIndex;comment:角色标识"`
	RoleSort  int       `json:"roleSort" gorm:"column:role_sort;type:int;not null;default:0;comment:显示顺序"`
	Status    string    `json:"status" gorm:"column:status;type:varchar(4);not null;default:2;comment:状态(1停用 2正常)"`
	Remark    string    `json:"remark" gorm:"column:remark;type:varchar(255);not null;default:'';comment:备注"`
	MenuIds   []int     `json:"menuIds" gorm:"-"`
	Menus     []SysMenu `json:"menus,omitempty" gorm:"many2many:sys_role_menu;joinForeignKey:RoleId;joinReferences:MenuId"`
	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at;type:datetime;not null;comment:创建时间"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"column:updated_at;type:datetime;not null;comment:更新时间"`
}

// TableName 表名。
func (SysRole) TableName() string { return enum.TableSysRole.String() }

// SysMenu 菜单/按钮权限。菜单树是前端动态路由的唯一来源。
type SysMenu struct {
	MenuId     int       `json:"menuId" gorm:"column:menu_id;type:int;not null;primaryKey;autoIncrement;comment:菜单ID"`
	MenuName   string    `json:"menuName" gorm:"column:menu_name;type:varchar(128);not null;default:'';comment:菜单名称(前端路由name)"`
	Title      string    `json:"title" gorm:"column:title;type:varchar(128);not null;default:'';comment:显示标题"`
	Icon       string    `json:"icon" gorm:"column:icon;type:varchar(128);not null;default:'';comment:图标"`
	Path       string    `json:"path" gorm:"column:path;type:varchar(128);not null;default:'';comment:路由地址"`
	MenuType   string    `json:"menuType" gorm:"column:menu_type;type:varchar(1);not null;default:'';comment:菜单类型(M目录 C页面 F按钮)"`
	Action     string    `json:"action" gorm:"column:action;type:varchar(16);not null;default:'';comment:操作标识"`
	Permission string    `json:"permission" gorm:"column:permission;type:varchar(255);not null;default:'';comment:权限标识(模块:资源:操作)"`
	ParentId   int       `json:"parentId" gorm:"column:parent_id;type:int;not null;default:0;index;comment:父菜单ID"`
	NoCache    bool      `json:"noCache" gorm:"column:no_cache;type:tinyint(1);not null;default:0;comment:是否不缓存(0否 1是)"`
	Component  string    `json:"component" gorm:"column:component;type:varchar(255);not null;default:'';comment:组件路径"`
	Sort       int       `json:"sort" gorm:"column:sort;type:int;not null;default:0;comment:显示顺序"`
	Visible    string    `json:"visible" gorm:"column:visible;type:varchar(1);not null;default:0;comment:显示状态(0显示 1隐藏)"`
	IsFrame    string    `json:"isFrame" gorm:"column:is_frame;type:varchar(1);not null;default:1;comment:是否外链(0是 1否)"`
	SysApi     []SysApi  `json:"sysApi,omitempty" gorm:"many2many:sys_menu_api_rule;joinForeignKey:MenuId;joinReferences:ApiId"`
	Apis       []int     `json:"apis" gorm:"-"`
	Children   []SysMenu `json:"children,omitempty" gorm:"-"`
	CreatedAt  time.Time `json:"createdAt" gorm:"column:created_at;type:datetime;not null;comment:创建时间"`
	UpdatedAt  time.Time `json:"updatedAt" gorm:"column:updated_at;type:datetime;not null;comment:更新时间"`
}

// TableName 表名。
func (SysMenu) TableName() string { return enum.TableSysMenu.String() }

// SysApi 后台接口登记表：菜单勾选接口后，角色据此获得放行规则。
type SysApi struct {
	ApiId     int       `json:"id" gorm:"column:api_id;type:int;not null;primaryKey;autoIncrement;comment:接口ID"`
	Path      string    `json:"path" gorm:"column:path;type:varchar(255);not null;default:'';index;comment:接口路径"`
	Handle    string    `json:"handle" gorm:"column:handle;type:varchar(255);not null;default:'';comment:处理方法"`
	Title     string    `json:"title" gorm:"column:title;type:varchar(255);not null;default:'';comment:接口名称"`
	Type      string    `json:"type" gorm:"column:type;type:varchar(8);not null;default:BUS;comment:接口类型(SYS基础设施 BUS业务)"`
	Action    string    `json:"action" gorm:"column:action;type:varchar(16);not null;default:'';comment:操作标识"`
	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at;type:datetime;not null;comment:创建时间"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"column:updated_at;type:datetime;not null;comment:更新时间"`
}

// TableName 表名。
func (SysApi) TableName() string { return enum.TableSysApi.String() }

// SysDictType 字典类型。
type SysDictType struct {
	DictId    int       `json:"id" gorm:"column:dict_id;type:int;not null;primaryKey;autoIncrement;comment:字典类型ID"`
	DictName  string    `json:"dictName" gorm:"column:dict_name;type:varchar(128);not null;default:'';comment:字典名称"`
	DictType  string    `json:"dictType" gorm:"column:dict_type;type:varchar(128);not null;default:'';uniqueIndex;comment:字典类型标识"`
	Status    string    `json:"status" gorm:"column:status;type:varchar(4);not null;default:2;comment:状态(1停用 2正常)"`
	Remark    string    `json:"remark" gorm:"column:remark;type:varchar(255);not null;default:'';comment:备注"`
	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at;type:datetime;not null;comment:创建时间"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"column:updated_at;type:datetime;not null;comment:更新时间"`
}

// TableName 表名。
func (SysDictType) TableName() string { return enum.TableSysDictType.String() }

// SysDictData 字典项。
type SysDictData struct {
	DictCode  int       `json:"dictCode" gorm:"column:dict_code;type:int;not null;primaryKey;autoIncrement;comment:字典项ID"`
	DictSort  int       `json:"dictSort" gorm:"column:dict_sort;type:int;not null;default:0;comment:显示顺序"`
	DictLabel string    `json:"dictLabel" gorm:"column:dict_label;type:varchar(128);not null;default:'';comment:字典标签"`
	DictValue string    `json:"dictValue" gorm:"column:dict_value;type:varchar(128);not null;default:'';comment:字典键值"`
	DictType  string    `json:"dictType" gorm:"column:dict_type;type:varchar(128);not null;default:'';index;comment:字典类型标识"`
	CssClass  string    `json:"cssClass" gorm:"column:css_class;type:varchar(128);not null;default:'';comment:样式类名"`
	ListClass string    `json:"listClass" gorm:"column:list_class;type:varchar(128);not null;default:'';comment:表格回显样式"`
	IsDefault string    `json:"isDefault" gorm:"column:is_default;type:varchar(8);not null;default:N;comment:是否默认(Y是 N否)"`
	Status    string    `json:"status" gorm:"column:status;type:varchar(4);not null;default:2;comment:状态(1停用 2正常)"`
	Remark    string    `json:"remark" gorm:"column:remark;type:varchar(255);not null;default:'';comment:备注"`
	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at;type:datetime;not null;comment:创建时间"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"column:updated_at;type:datetime;not null;comment:更新时间"`
}

// TableName 表名。
func (SysDictData) TableName() string { return enum.TableSysDictData.String() }

// SysConfig 系统配置项。
type SysConfig struct {
	ConfigId    int       `json:"id" gorm:"column:config_id;type:int;not null;primaryKey;autoIncrement;comment:配置ID"`
	ConfigName  string    `json:"configName" gorm:"column:config_name;type:varchar(128);not null;default:'';comment:配置名称"`
	ConfigKey   string    `json:"configKey" gorm:"column:config_key;type:varchar(128);not null;default:'';uniqueIndex;comment:配置键"`
	ConfigValue string    `json:"configValue" gorm:"column:config_value;type:varchar(1024);not null;default:'';comment:配置值"`
	ConfigType  string    `json:"configType" gorm:"column:config_type;type:varchar(8);not null;default:N;comment:配置类型(Y内置 N自定义)"`
	IsFrontend  string    `json:"isFrontend" gorm:"column:is_frontend;type:varchar(8);not null;default:0;comment:是否前端使用(0否 1是)"`
	Remark      string    `json:"remark" gorm:"column:remark;type:varchar(255);not null;default:'';comment:备注"`
	CreatedAt   time.Time `json:"createdAt" gorm:"column:created_at;type:datetime;not null;comment:创建时间"`
	UpdatedAt   time.Time `json:"updatedAt" gorm:"column:updated_at;type:datetime;not null;comment:更新时间"`
}

// TableName 表名。
func (SysConfig) TableName() string { return enum.TableSysConfig.String() }

// SysLoginLog 登录审计。
type SysLoginLog struct {
	Id            int       `json:"id" gorm:"column:info_id;type:int;not null;primaryKey;autoIncrement;comment:日志ID"`
	Username      string    `json:"username" gorm:"column:username;type:varchar(64);not null;default:'';index;comment:登录账号"`
	Ipaddr        string    `json:"ipaddr" gorm:"column:ipaddr;type:varchar(64);not null;default:'';comment:登录IP"`
	LoginLocation string    `json:"loginLocation" gorm:"column:login_location;type:varchar(128);not null;default:'';comment:登录地点"`
	Browser       string    `json:"browser" gorm:"column:browser;type:varchar(128);not null;default:'';comment:浏览器"`
	Os            string    `json:"os" gorm:"column:os;type:varchar(128);not null;default:'';comment:操作系统"`
	Platform      string    `json:"platform" gorm:"column:platform;type:varchar(128);not null;default:'';comment:平台"`
	Status        string    `json:"status" gorm:"column:status;type:varchar(4);not null;default:'';comment:状态(1失败 2成功)"`
	Msg           string    `json:"msg" gorm:"column:msg;type:varchar(255);not null;default:'';comment:提示信息"`
	LoginTime     time.Time `json:"loginTime" gorm:"column:login_time;type:datetime;not null;comment:登录时间"`
	CreatedAt     time.Time `json:"createdAt" gorm:"column:created_at;type:datetime;not null;comment:创建时间"`
}

// TableName 表名。
func (SysLoginLog) TableName() string { return enum.TableSysLoginLog.String() }

// SysOperaLog 操作审计。
type SysOperaLog struct {
	Id            int       `json:"id" gorm:"column:oper_id;type:int;not null;primaryKey;autoIncrement;comment:日志ID"`
	Title         string    `json:"title" gorm:"column:title;type:varchar(128);not null;default:'';comment:操作模块"`
	BusinessType  string    `json:"businessType" gorm:"column:business_type;type:varchar(8);not null;default:'';comment:业务类型"`
	Method        string    `json:"method" gorm:"column:method;type:varchar(255);not null;default:'';comment:处理方法"`
	RequestMethod string    `json:"requestMethod" gorm:"column:request_method;type:varchar(16);not null;default:'';comment:请求方式"`
	OperUrl       string    `json:"operUrl" gorm:"column:oper_url;type:varchar(255);not null;default:'';comment:请求地址"`
	OperIp        string    `json:"operIp" gorm:"column:oper_ip;type:varchar(64);not null;default:'';comment:操作IP"`
	OperName      string    `json:"operName" gorm:"column:oper_name;type:varchar(64);not null;default:'';comment:操作人"`
	Status        string    `json:"status" gorm:"column:status;type:varchar(4);not null;default:'';comment:状态(1失败 2成功)"`
	OperTime      time.Time `json:"operTime" gorm:"column:oper_time;type:datetime;not null;comment:操作时间"`
	OperLocation  string    `json:"operLocation" gorm:"column:oper_location;type:varchar(128);not null;default:'';comment:操作地点"`
	OperParam     string    `json:"operParam" gorm:"column:oper_param;type:varchar(2048);not null;default:'';comment:请求参数"`
	JsonResult    string    `json:"jsonResult" gorm:"column:json_result;type:varchar(2048);not null;default:'';comment:返回结果"`
	LatencyTime   string    `json:"latencyTime" gorm:"column:latency_time;type:varchar(16);not null;default:'';comment:耗时"`
	CreatedAt     time.Time `json:"createdAt" gorm:"column:created_at;type:datetime;not null;comment:创建时间"`
}

// TableName 表名。
func (SysOperaLog) TableName() string { return enum.TableSysOperaLog.String() }

// SysCasbinRule 放行规则：ptype=p 时 v0=roleKey v1=path v2=method。
type SysCasbinRule struct {
	Id    int    `json:"id" gorm:"column:id;type:int;not null;primaryKey;autoIncrement;comment:规则ID"`
	Ptype string `json:"ptype" gorm:"column:ptype;type:varchar(16);not null;default:'';index;comment:规则类型"`
	V0    string `json:"v0" gorm:"column:v0;type:varchar(128);not null;default:'';comment:规则值0(角色标识)"`
	V1    string `json:"v1" gorm:"column:v1;type:varchar(255);not null;default:'';comment:规则值1(接口路径)"`
	V2    string `json:"v2" gorm:"column:v2;type:varchar(16);not null;default:'';comment:规则值2(请求方式)"`
	V3    string `json:"v3" gorm:"column:v3;type:varchar(255);not null;default:'';comment:规则值3"`
	V4    string `json:"v4" gorm:"column:v4;type:varchar(255);not null;default:'';comment:规则值4"`
	V5    string `json:"v5" gorm:"column:v5;type:varchar(255);not null;default:'';comment:规则值5"`
}

// TableName 表名。
func (SysCasbinRule) TableName() string { return enum.TableSysCasbinRule.String() }
