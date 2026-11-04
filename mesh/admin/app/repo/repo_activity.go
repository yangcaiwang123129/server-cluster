package repo

import (
	"gorm.io/gorm"

	"server-cluster/share/consts"
	"server-cluster/share/model"
	"server-cluster/share/util"
)

// ActivityQuery 活动实例列表过滤条件。
type ActivityQuery struct {
	CfgID  int32
	Status int32 // consts.ActivityStatusAll 表示全部
	Order  string
}

// ListActivities 分页查询活动实例。
func (r *Repo) ListActivities(q ActivityQuery, pageIndex, pageSize int) ([]model.Activity, int64, error) {
	d, err := r.db()
	if err != nil {
		return nil, 0, err
	}
	tx := d.Model(&model.Activity{})
	if q.CfgID > 0 {
		tx = tx.Where("cfg_id = ?", q.CfgID)
	}
	if q.Status != consts.ActivityStatusAll {
		tx = tx.Where("status = ?", q.Status)
	}
	return util.PageQuery[model.Activity](tx, pageIndex, pageSize, q.Order, "id desc")
}

// GetActivity 按主键查询活动实例。
func (r *Repo) GetActivity(id int64) (*model.Activity, error) {
	d, err := r.db()
	if err != nil {
		return nil, err
	}
	var m model.Activity
	if err := d.First(&m, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// CreateActivity 新增活动实例（自动生成）。
func (r *Repo) CreateActivity(m *model.Activity) error {
	d, err := r.db()
	if err != nil {
		return err
	}
	if m.Round <= 0 {
		m.Round = 1
	}
	return d.Create(m).Error
}

// UpdateActivityTime 修改活动时间；仅未开启允许。
func (r *Repo) UpdateActivityTime(id, startTime, endTime int64) (bool, error) {
	d, err := r.db()
	if err != nil {
		return false, err
	}
	now := util.NowUnixMilli()
	res := d.Model(&model.Activity{}).
		Where("id = ? AND status = ?", id, consts.ActivityStatusNotStarted).
		Updates(map[string]any{
			"start_time": startTime,
			"end_time":   endTime,
			"updated_at": now,
			"version":    gorm.Expr("version + 1"),
		})
	return res.RowsAffected > 0, res.Error
}

// TransitionActivityStatus 条件更新状态（CAS），返回是否命中。
func (r *Repo) TransitionActivityStatus(id int64, from, to int32) (bool, error) {
	d, err := r.db()
	if err != nil {
		return false, err
	}
	now := util.NowUnixMilli()
	res := d.Model(&model.Activity{}).
		Where("id = ? AND status = ?", id, from).
		Updates(map[string]any{
			"status":     to,
			"updated_at": now,
			"version":    gorm.Expr("version + 1"),
		})
	return res.RowsAffected > 0, res.Error
}

// ListActivitiesByStatus 查询指定状态的全部活动实例。
func (r *Repo) ListActivitiesByStatus(status int32) ([]model.Activity, error) {
	d, err := r.db()
	if err != nil {
		return nil, err
	}
	var list []model.Activity
	err = d.Where("status = ?", status).Order("id asc").Find(&list).Error
	return list, err
}

// ListAllActivities 查询全量活动实例（活动模块启动恢复用）。
func (r *Repo) ListAllActivities() ([]model.Activity, error) {
	d, err := r.db()
	if err != nil {
		return nil, err
	}
	var list []model.Activity
	err = d.Order("id asc").Find(&list).Error
	return list, err
}
