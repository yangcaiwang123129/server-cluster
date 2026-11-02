package dto

// ClusterNodeQuery 集群节点列表查询参数（分页接口）。
type ClusterNodeQuery struct {
	NodeName  string `json:"nodeName"`
	NodeKind  string `json:"nodeKind"`
	Online    *int   `json:"online"`
	Order     string `json:"order"`
	PageIndex int    `json:"pageIndex"`
	PageSize  int    `json:"pageSize"`
}

// ClusterNodeResp 集群节点详情（来自 gm_cluster_node 表，已含在线同步）。
type ClusterNodeResp struct {
	NodeID     string   `json:"nodeId"`
	NodeName   string   `json:"nodeName"`
	NodeKind   string   `json:"nodeKind"`
	Alias      string   `json:"alias"`
	Endpoint   string   `json:"endpoint"`
	Services   []string `json:"services"`
	Online     int8     `json:"online"`
	SSHHost    string   `json:"sshHost"`
	SSHPort    int32    `json:"sshPort"`
	SSHUser    string   `json:"sshUser"`
	SSHKeyPath string   `json:"sshKeyPath"`
	DeployDir  string   `json:"deployDir"`
	Remark     string   `json:"remark"`
	CreatedAt  string   `json:"createdAt"`
	UpdatedAt  string   `json:"updatedAt"`
}

// ClusterNodeUpdateReq 更新运维侧 SSH 配置（只允许修改运维字段，在线/endpoint 等不动）。
type ClusterNodeUpdateReq struct {
	SSHHost    string `json:"sshHost"`
	SSHPort    int32  `json:"sshPort"`
	SSHUser    string `json:"sshUser"`
	SSHKeyPath string `json:"sshKeyPath"`
	DeployDir  string `json:"deployDir"`
	Remark     string `json:"remark"`
}

// ClusterExecReq 远程执行脚本请求。
// action: start / stop / restart / deploy。
type ClusterExecReq struct {
	Action string `json:"action"`
}

// ClusterExecResp 远程执行脚本完整回显。
type ClusterExecResp struct {
	NodeID   string `json:"nodeId"`
	Action   string `json:"action"`
	ExitCode int    `json:"exitCode"` // -1 表示 SSH/网络层错误（此时 Stderr 含错误信息）
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}
