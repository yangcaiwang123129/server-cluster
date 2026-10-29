package repo

import (
	"server-cluster/share/model"
	"server-cluster/share/util"
)

// ClientVersionQuery 客户端版本列表过滤条件。
type ClientVersionQuery struct {
	Version  string
	Platform string
	Status   string
	Order    string
}

// ListClientVersions 分页查询客户端版本。
func (r *Repo) ListClientVersions(q ClientVersionQuery, pageIndex, pageSize int) ([]model.ClientVersion, int64, error) {
	d, err := r.db()
	if err != nil {
		return nil, 0, err
	}
	tx := d.Model(&model.ClientVersion{})
	if q.Version != "" {
		tx = tx.Where("version LIKE ?", util.Like(q.Version))
	}
	if q.Platform != "" {
		tx = tx.Where("platform = ?", q.Platform)
	}
	if q.Status != "" {
		tx = tx.Where("status = ?", q.Status)
	}
	return util.PageQuery[model.ClientVersion](tx, pageIndex, pageSize, q.Order, "version_id desc")
}

// GetClientVersion 按主键查询客户端版本。
func (r *Repo) GetClientVersion(id int) (*model.ClientVersion, error) {
	d, err := r.db()
	if err != nil {
		return nil, err
	}
	var m model.ClientVersion
	if err := d.First(&m, "version_id = ?", id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// CreateClientVersion 新增客户端版本。
func (r *Repo) CreateClientVersion(m *model.ClientVersion) error {
	d, err := r.db()
	if err != nil {
		return err
	}
	return d.Create(m).Error
}

// UpdateClientVersion 修改客户端版本。
func (r *Repo) UpdateClientVersion(m *model.ClientVersion) error {
	d, err := r.db()
	if err != nil {
		return err
	}
	return d.Model(&model.ClientVersion{}).Where("version_id = ?", m.VersionId).
		Select("version", "platform", "download_url", "force_update", "status", "release_note").
		Updates(m).Error
}

// DeleteClientVersions 批量删除客户端版本。
func (r *Repo) DeleteClientVersions(ids []int) error {
	d, err := r.db()
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	return d.Where("version_id IN ?", ids).Delete(&model.ClientVersion{}).Error
}

// UpdateClientVersionStatus 启停客户端版本。
func (r *Repo) UpdateClientVersionStatus(id int, status string) error {
	d, err := r.db()
	if err != nil {
		return err
	}
	return d.Model(&model.ClientVersion{}).Where("version_id = ?", id).
		Update("status", status).Error
}

// ExistsClientVersion 同一平台下是否已存在该版本号；excludeId>0 时排除自身。
func (r *Repo) ExistsClientVersion(version, platform string, excludeId int) (bool, error) {
	d, err := r.db()
	if err != nil {
		return false, err
	}
	tx := d.Model(&model.ClientVersion{}).Where("version = ? AND platform = ?", version, platform)
	if excludeId > 0 {
		tx = tx.Where("version_id <> ?", excludeId)
	}
	var count int64
	if err := tx.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
