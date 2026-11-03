package model

import (
	"server-cluster/share/enum"
	"time"
)

// ClusterNode 集群节点注册表（运维视角）。
//
// 主键 node_id 对齐 due 服务注册中心的 ServiceInstance.ID。
// 列表接口每次查询时都会把 etcd 里当前在线的节点 upsert 进本表：
//   - 表里没有 → 插入
//   - 表里有 → 更新 online/endpoint/services 等动态字段
//
// SSHHost/SSHUser/SSHKeyPath/DeployDir 等运维配置字段**不会被 etcd 同步覆盖**，
// 需要管理员在列表里手动录入（或通过其他运维工具批量写入）。
// 主键 node_id 是 due 在 etcd 里唯一的实例 ID，全集群不会重复，
// 但为了可读性同时记录 node_kind（gate/node/mesh/master）。
type ClusterNode struct {
	NodeID     string    `gorm:"column:node_id;type:varchar(128);not null;primaryKey;comment:实例ID(etcd key的最后一段)" json:"nodeId"`
	NodeName   string    `gorm:"column:node_name;type:varchar(64);not null;default:'';index;comment:服务名(gate/node/mesh)" json:"nodeName"`
	NodeKind   string    `gorm:"column:node_kind;type:varchar(16);not null;default:'';comment:服务类型(gate/node/mesh/master)" json:"nodeKind"`
	Alias      string    `gorm:"column:alias;type:varchar(64);not null;default:'';comment:实例别名(如gate-1 game-1)" json:"alias"`
	Endpoint   string    `gorm:"column:endpoint;type:varchar(128);not null;default:'';comment:内部服务端点" json:"endpoint"`
	Services   string    `gorm:"column:services;type:varchar(1024);not null;default:'';comment:mesh实例暴露的内部服务列表(逗号分隔)" json:"services"`
	Online     int8      `gorm:"column:online;type:tinyint;not null;default:0;index;comment:在线标记(1在etcd中 0已离线)" json:"online"`
	SSHHost    string    `gorm:"column:ssh_host;type:varchar(128);not null;default:'';comment:SSH目标主机(可IP或域名)" json:"sshHost"`
	SSHPort    int32     `gorm:"column:ssh_port;type:int;not null;default:22;comment:SSH端口" json:"sshPort"`
	SSHUser    string    `gorm:"column:ssh_user;type:varchar(64);not null;default:'';comment:SSH用户名" json:"sshUser"`
	SSHKeyPath string    `gorm:"column:ssh_key_path;type:varchar(256);not null;default:'';comment:本机私钥路径" json:"sshKeyPath"`
	DeployDir  string    `gorm:"column:deploy_dir;type:varchar(256);not null;default:'';comment:服务器上的部署目录(含start.sh/stop.sh)" json:"deployDir"`
	Remark     string    `gorm:"column:remark;type:varchar(255);not null;default:'';comment:备注" json:"remark"`
	CreatedAt  time.Time `gorm:"column:created_at;type:datetime;not null;comment:首次入库时间" json:"createdAt"`
	UpdatedAt  time.Time `gorm:"column:updated_at;type:datetime;not null;comment:最后同步/更新时间" json:"updatedAt"`
}

// TableName 表名。
func (ClusterNode) TableName() string { return enum.TableCluster.String() }
