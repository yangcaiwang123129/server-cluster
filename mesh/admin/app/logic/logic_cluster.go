package logic

import (
	"context"
	"fmt"
	"strings"
	"time"

	"server-cluster/internal/svc"
	"server-cluster/mesh/admin/app/global"
	"server-cluster/mesh/admin/app/repo"
	"server-cluster/share/dto"
	"server-cluster/share/model"
	"server-cluster/share/singleton"
	"server-cluster/share/util"

	"github.com/dobyte/due/v2/cluster"
	"github.com/dobyte/due/v2/log"
	"github.com/dobyte/due/v2/registry"
)

var clusterLogicInst = singleton.NewContainer[global.ClusterLogic]("ClusterLogic")

func RegisterClusterLogic(inst global.ClusterLogic) { clusterLogicInst.Register(inst) }

// Cluster 获取集群管理业务单例。
func Cluster() global.ClusterLogic { return clusterLogicInst.MustGet() }

func init() { RegisterClusterLogic(&clusterLogic{}) }

type clusterLogic struct{}

// 要枚举的 due 服务名：gate / node / mesh。
// master 当前项目未使用，但预留以防未来需要。
var clusterKinds = []string{cluster.Gate.String(), cluster.Node.String(), cluster.Mesh.String()}

func (this *clusterLogic) Sync(ctx context.Context) error {
	reg := svc.Ctx().Registry
	if reg == nil {
		log.Warnf("admin: registry not ready, skip cluster sync")
		return nil
	}
	type pair struct {
		list []*registry.ServiceInstance
		err  error
	}
	ch := make(chan pair, len(clusterKinds))
	for _, kind := range clusterKinds {
		go func(name string) {
			list, err := reg.Services(ctx, name)
			ch <- pair{list, err}
		}(kind)
	}
	merged := make([]*registry.ServiceInstance, 0)
	seen := make(map[string]struct{})
	for range clusterKinds {
		p := <-ch
		if p.err != nil {
			log.Warnf("admin: sync cluster kind err: %v", p.err)
			continue
		}
		for _, ins := range p.list {
			if ins == nil || ins.ID == "" {
				continue
			}
			if _, ok := seen[ins.ID]; ok {
				continue
			}
			seen[ins.ID] = struct{}{}
			merged = append(merged, ins)
		}
	}
	return repo.R().SyncClusterNodes(merged)
}

func (this *clusterLogic) List(ctx context.Context, q dto.ClusterNodeQuery, pageIndex, pageSize int) ([]dto.ClusterNodeResp, int64, error) {
	// 同步失败不阻塞查询，直接查 DB 现状，但必须留下错误日志便于定位
	if err := this.Sync(ctx); err != nil {
		log.Warnf("admin: sync cluster nodes failed: %v", err)
	}
	list, total, err := repo.R().ListClusterNodes(clusterNodeQuery(q), pageIndex, pageSize)
	if err != nil {
		return nil, 0, err
	}
	return toClusterNodeRespList(list), total, nil
}

func (this *clusterLogic) Get(ctx context.Context, nodeID string) (*dto.ClusterNodeResp, error) {
	m, err := repo.R().GetClusterNode(nodeID)
	if err != nil {
		return nil, err
	}
	// 详情接口顺带触发一次 Sync（失败只记日志），刷新 online 标记
	if err := this.Sync(ctx); err != nil {
		log.Warnf("admin: sync cluster nodes failed: %v", err)
	}
	return toClusterNodeResp(m), nil
}

func (this *clusterLogic) UpdateSSH(_ context.Context, m *model.ClusterNode) error {
	return repo.R().UpdateClusterNodeSSH(m)
}

func (this *clusterLogic) Exec(ctx context.Context, nodeID, action string) (*dto.ClusterExecResp, error) {
	node, err := repo.R().GetClusterNode(nodeID)
	if err != nil {
		return nil, err
	}
	sshCfg := svc.SSH().ResolveFromEndpoint(node.Endpoint)
	if sshCfg == nil {
		return nil, fmt.Errorf("node %s (endpoint=%s) 未在 etc.toml 的 [[ssh.hosts]] 里配置 SSH", nodeID, node.Endpoint)
	}
	if sshCfg.User == "" {
		return nil, fmt.Errorf("node %s: ssh.user 未配置", nodeID)
	}
	// deployDir = defaultDeployDir + "/" + alias（alias 即服务实例短名，如 "game" / "gate"）
	deployDir := sshCfg.DefaultDeployDir
	if deployDir == "" {
		return nil, fmt.Errorf("node %s: ssh.defaultDeployDir 未配置", nodeID)
	}
	if node.Alias != "" {
		deployDir = strings.TrimRight(deployDir, "/") + "/" + node.Alias
	}
	var cmd string
	// 拼接 shell：cd 到 deploy_dir + exec 对应脚本
	switch strings.ToLower(action) {
	case "start", "deploy":
		cmd = fmt.Sprintf("cd '%s' && bash start.sh", deployDir)
	case "stop":
		cmd = fmt.Sprintf("cd '%s' && bash stop.sh", deployDir)
	case "restart", "hotreload":
		// stop.sh 已经内置优雅停服 + 资源释放等待 + 兜底强杀，之后直接 start.sh 即可
		cmd = fmt.Sprintf("cd '%s' && bash stop.sh && bash start.sh", deployDir)
	default:
		return nil, fmt.Errorf("unknown action: %s (start/stop/restart/deploy)", action)
	}
	cfg := util.SSHConfig{
		Host:     sshCfg.Host,
		Port:     sshCfg.Port,
		User:     sshCfg.User,
		Password: sshCfg.Password,
		KeyPath:  sshCfg.KeyPath,
		Timeout:  3 * time.Minute, // stop.sh 默认等待 20s + EXIT_TIMEOUT + start.sh 0.5s sleep
	}
	result, err := util.RunSSH(ctx, cfg, cmd)
	if err != nil {
		return &dto.ClusterExecResp{
			NodeID:   nodeID,
			Action:   action,
			ExitCode: -1,
			Stdout:   "",
			Stderr:   err.Error(),
		}, nil
	}
	return &dto.ClusterExecResp{
		NodeID:   nodeID,
		Action:   action,
		ExitCode: result.ExitCode,
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
	}, nil
}

// ── DTO 转换 ──────────────────────────────────────────────

func clusterNodeQuery(q dto.ClusterNodeQuery) repo.ClusterNodeQuery {
	rq := repo.ClusterNodeQuery{
		NodeName: q.NodeName,
		NodeKind: q.NodeKind,
		Order:    q.Order,
	}
	if q.Online != nil {
		v := int(*q.Online)
		rq.Online = &v
	}
	return rq
}

func toClusterNodeRespList(list []model.ClusterNode) []dto.ClusterNodeResp {
	return util.MapSlice(list, func(m model.ClusterNode) dto.ClusterNodeResp {
		return *toClusterNodeResp(&m)
	})
}

func toClusterNodeResp(m *model.ClusterNode) *dto.ClusterNodeResp {
	return &dto.ClusterNodeResp{
		NodeID:     m.NodeID,
		NodeName:   m.NodeName,
		NodeKind:   m.NodeKind,
		Alias:      m.Alias,
		Endpoint:   m.Endpoint,
		Services:   util.SplitCSV(m.Services),
		Online:     m.Online,
		SSHHost:    m.SSHHost,
		SSHPort:    m.SSHPort,
		SSHUser:    m.SSHUser,
		SSHKeyPath: m.SSHKeyPath,
		DeployDir:  m.DeployDir,
		Remark:     m.Remark,
		CreatedAt:  m.CreatedAt.Format(util.DateTime),
		UpdatedAt:  m.UpdatedAt.Format(util.DateTime),
	}
}
