package repo

import (
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"server-cluster/internal/svc"
	"server-cluster/share/model"
	"server-cluster/share/util"

	"github.com/dobyte/due/v2/registry"
)

// ClusterNodeQuery 集群节点列表过滤条件。
type ClusterNodeQuery struct {
	NodeName string
	NodeKind string
	Online   *int
	Order    string
}

// ListClusterNodes 分页查询集群节点（不触发 etcd 同步）。
func (r *Repo) ListClusterNodes(q ClusterNodeQuery, pageIndex, pageSize int) ([]model.ClusterNode, int64, error) {
	d, err := r.db()
	if err != nil {
		return nil, 0, err
	}
	tx := d.Model(&model.ClusterNode{})
	if q.NodeName != "" {
		tx = tx.Where("node_name LIKE ?", util.Like(q.NodeName))
	}
	if q.NodeKind != "" {
		tx = tx.Where("node_kind = ?", q.NodeKind)
	}
	if q.Online != nil {
		tx = tx.Where("online = ?", *q.Online)
	}
	return util.PageQuery[model.ClusterNode](tx, pageIndex, pageSize, q.Order, "node_kind asc, node_name asc")
}

// GetClusterNode 按主键查询集群节点。
func (r *Repo) GetClusterNode(nodeID string) (*model.ClusterNode, error) {
	d, err := r.db()
	if err != nil {
		return nil, err
	}
	var m model.ClusterNode
	if err := d.First(&m, "node_id = ?", nodeID).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// UpdateClusterNodeSSH 更新运维侧 SSH 连接配置；动态字段（online/endpoint 等）不允许手动改。
func (r *Repo) UpdateClusterNodeSSH(m *model.ClusterNode) error {
	d, err := r.db()
	if err != nil {
		return err
	}
	return d.Model(&model.ClusterNode{}).Where("node_id = ?", m.NodeID).
		Select("ssh_host", "ssh_port", "ssh_user", "ssh_key_path", "deploy_dir", "remark").
		Updates(m).Error
}

// SyncClusterNodes 从 etcd registry 同步集群节点到 DB：批量 Upsert 在线节点，本次未出现的置 offline，不覆盖 SSH 运维配置字段。
func (r *Repo) SyncClusterNodes(instances []*registry.ServiceInstance) error {
	d, err := r.db()
	if err != nil {
		return err
	}
	now := util.Now()
	// 1. 构造 Upsert 数据；只写动态字段
	rows := make([]map[string]any, 0, len(instances))
	for _, ins := range instances {
		if ins == nil || ins.ID == "" {
			continue
		}
		services := strings.Join(ins.Services, ",")
		rows = append(rows, map[string]any{
			"node_id":    ins.ID,
			"node_name":  ins.Name,
			"node_kind":  ins.Kind,
			"alias":      ins.Alias,
			"endpoint":   ins.Endpoint,
			"services":   services,
			"online":     int8(1),
			"updated_at": now,
			"created_at": now, // Upsert 冲突时不会覆盖 created_at
		})
	}
	if len(rows) > 0 {
		if err := d.Table(model.ClusterNode{}.TableName()).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "node_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"node_name", "node_kind", "alias", "endpoint", "services", "online", "updated_at"}),
		}).Create(&rows).Error; err != nil {
			return err
		}
	}
	// 2. 把这次没出现的、但 DB 里还 online=1 的节点 → 置 offline
	if err := r.markOffline(d, rows); err != nil {
		return err
	}
	return nil
}

// markOffline 把本次 etcd 里不再存在的节点置 online=0。
func (r *Repo) markOffline(d *gorm.DB, rows []map[string]any) error {
	if len(rows) == 0 {
		// etcd 里一个节点都没有：所有 DB 记录都应该置 offline
		return d.Model(&model.ClusterNode{}).Where("online = ?", 1).
			Updates(map[string]any{"online": int8(0), "updated_at": util.Now()}).Error
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		if id, ok := row["node_id"].(string); ok && id != "" {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	return d.Model(&model.ClusterNode{}).
		Where("online = ? AND node_id NOT IN ?", 1, ids).
		Updates(map[string]any{"online": int8(0), "updated_at": util.Now()}).Error
}

// EnsureClusterNodesTable 确保集群节点表存在（admin AutoMigrate 已覆盖，这里兜底）。
func (r *Repo) EnsureClusterNodesTable() error {
	d, err := r.db()
	if err != nil {
		return err
	}
	return svc.AutoMigrate(d, &model.ClusterNode{})
}
