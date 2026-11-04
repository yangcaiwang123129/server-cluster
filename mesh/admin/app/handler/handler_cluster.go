package handler

import (
	"server-cluster/share/util"
	"strconv"
	"strings"

	"server-cluster/mesh/admin/app/logic"
	"server-cluster/share/dto"
	"server-cluster/share/errs"
	"server-cluster/share/model"

	"github.com/dobyte/due/component/http/v2"
)

// RegisterCluster 注册集群管理路由（列表/详情/SSH配置更新/远程运维）。
func RegisterCluster(router http.Router) {
	g := gmGroup(router)
	GET(g, "/cluster", handleClusterList)
	GET(g, "/cluster/:nodeId", handleClusterDetail)
	PUT(g, "/cluster/:nodeId", handleClusterUpdate)
	POST(g, "/cluster/:nodeId/exec", handleClusterExec)
}

// handleClusterList 集群节点分页列表（自动触发 etcd → DB 同步）。
// @Summary 集群节点列表
// @Tags cluster
// @Produce json
// @Param nodeName query string false "服务名(gate/node/mesh)"
// @Param nodeKind query string false "服务类型(gate/node/mesh)"
// @Param online query int false "在线筛选(1在线 0离线)"
// @Param pageIndex query int false "页码" default(1)
// @Param pageSize query int false "每页条数" default(10)
// @Router /api/v1/gm/cluster [get]
func handleClusterList(ctx http.Context) (util.PageResp[dto.ClusterNodeResp], error) {
	var req dto.ClusterNodeQuery
	req.NodeName = ctx.Query("nodeName")
	req.NodeKind = ctx.Query("nodeKind")
	if v := ctx.Query("online"); v != "" {
		if i, err := strconv.Atoi(v); err == nil && (i == 0 || i == 1) {
			vv := i
			req.Online = &vv
		}
	}
	req.PageIndex, req.PageSize = pageParams(ctx)
	req.Order = orderBy(ctx, map[string]string{
		"nodeId":   "node_id",
		"nodeName": "node_name",
		"nodeKind": "node_kind",
		"online":   "online",
	})
	list, total, err := logic.Cluster().List(ctx.Context(), req, req.PageIndex, req.PageSize)
	if err != nil {
		return util.PageResp[dto.ClusterNodeResp]{}, err
	}
	return util.Page(list, total, req.PageIndex, req.PageSize), nil
}

// handleClusterDetail 单个集群节点详情（带完整 SSH 配置）。
// @Summary 集群节点详情
// @Tags cluster
// @Produce json
// @Param nodeId path string true "实例ID(etcd key最后一段)"
// @Router /api/v1/gm/cluster/{nodeId} [get]
func handleClusterDetail(ctx http.Context) (*dto.ClusterNodeResp, error) {
	nodeID := ctx.Params("nodeId")
	if nodeID == "" {
		return nil, errs.NewError(errs.RespCodeInternal, "nodeId required")
	}
	return logic.Cluster().Get(ctx.Context(), nodeID)
}

// handleClusterUpdate 更新运维侧 SSH 配置。
// @Summary 更新节点 SSH 配置
// @Tags cluster
// @Produce json
// @Param nodeId path string true "实例ID"
// @Param body body dto.ClusterNodeUpdateReq true "SSH 配置"
// @Router /api/v1/gm/cluster/{nodeId} [put]
func handleClusterUpdate(ctx http.Context) (any, error) {
	nodeID := ctx.Params("nodeId")
	if nodeID == "" {
		return nil, errs.NewError(errs.RespCodeInternal, "nodeId required")
	}
	var req dto.ClusterNodeUpdateReq
	if err := bindBody(ctx, &req); err != nil {
		return nil, errs.NewError(errs.RespCodeInternal, "invalid body")
	}
	m := &model.ClusterNode{
		NodeID:     nodeID,
		SSHHost:    strings.TrimSpace(req.SSHHost),
		SSHPort:    req.SSHPort,
		SSHUser:    strings.TrimSpace(req.SSHUser),
		SSHKeyPath: strings.TrimSpace(req.SSHKeyPath),
		DeployDir:  strings.TrimSpace(req.DeployDir),
		Remark:     strings.TrimSpace(req.Remark),
	}
	if m.SSHHost == "" || m.SSHUser == "" {
		return nil, errs.NewError(errs.RespCodeInternal, "sshHost and sshUser required")
	}
	return nil, logic.Cluster().UpdateSSH(ctx.Context(), m)
}

// handleClusterExec 远程执行运维脚本并回显完整输出。
// @Summary 远程运维脚本执行
// @Tags cluster
// @Produce json
// @Param nodeId path string true "实例ID"
// @Param body body dto.ClusterExecReq true "{action:start|stop|restart|deploy}"
// @Router /api/v1/gm/cluster/{nodeId}/exec [post]
func handleClusterExec(ctx http.Context) (*dto.ClusterExecResp, error) {
	nodeID := ctx.Params("nodeId")
	if nodeID == "" {
		return nil, errs.NewError(errs.RespCodeInternal, "nodeId required")
	}
	var req dto.ClusterExecReq
	if err := bindBody(ctx, &req); err != nil {
		return nil, errs.NewError(errs.RespCodeInternal, "invalid body")
	}
	action := strings.ToLower(strings.TrimSpace(req.Action))
	switch action {
	case "start", "stop", "restart", "deploy":
	default:
		return nil, errs.NewError(errs.RespCodeInternal, "action must be start/stop/restart/deploy")
	}
	return logic.Cluster().Exec(ctx.Context(), nodeID, action)
}
