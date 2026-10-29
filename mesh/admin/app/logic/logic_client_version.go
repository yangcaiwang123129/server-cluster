package logic

import (
	"strings"

	"server-cluster/mesh/admin/app/repo"
	"server-cluster/share/consts"
	"server-cluster/share/dto"
	"server-cluster/share/errs"
	"server-cluster/share/model"
)

// ClientVersionList 客户端版本分页列表。
func ClientVersionList(req dto.ClientVersionListReq) ([]model.ClientVersion, int64, error) {
	return repo.R().ListClientVersions(repo.ClientVersionQuery{
		Version:  req.Version,
		Platform: req.Platform,
		Status:   req.Status,
		Order:    req.Order,
	}, req.PageIndex, req.PageSize)
}

// ClientVersionGet 客户端版本详情。
func ClientVersionGet(id int) (*model.ClientVersion, error) {
	return repo.R().GetClientVersion(id)
}

// ClientVersionCreate 新增客户端版本。
func ClientVersionCreate(req dto.ClientVersionSaveReq) (*model.ClientVersion, error) {
	m, err := clientVersionModel(req, 0)
	if err != nil {
		return nil, err
	}
	if err := repo.R().CreateClientVersion(m); err != nil {
		return nil, err
	}
	return m, nil
}

// ClientVersionUpdate 修改客户端版本。
func ClientVersionUpdate(id int, req dto.ClientVersionSaveReq) (*model.ClientVersion, error) {
	if id == 0 {
		return nil, errs.NewError(400, "版本ID不能为空")
	}
	m, err := clientVersionModel(req, id)
	if err != nil {
		return nil, err
	}
	m.VersionId = id
	if err := repo.R().UpdateClientVersion(m); err != nil {
		return nil, err
	}
	// 重新读一次，返回带创建/更新时间的完整记录。
	return repo.R().GetClientVersion(id)
}

// ClientVersionDelete 批量删除客户端版本。
func ClientVersionDelete(ids []int) error { return repo.R().DeleteClientVersions(ids) }

// ClientVersionUpdateStatus 启用/停用客户端版本。
func ClientVersionUpdateStatus(req dto.ClientVersionStatusReq) error {
	if req.VersionId == 0 {
		return errs.NewError(400, "版本ID不能为空")
	}
	return repo.R().UpdateClientVersionStatus(req.VersionId, req.Status)
}

// clientVersionModel 校验请求并转模型；excludeId 用于修改时排除自身的重号检查。
func clientVersionModel(req dto.ClientVersionSaveReq, excludeId int) (*model.ClientVersion, error) {
	version := strings.TrimSpace(req.Version)
	platform := strings.TrimSpace(req.Platform)
	if version == "" {
		return nil, errs.NewError(400, "版本号不能为空")
	}
	if platform == "" {
		return nil, errs.NewError(400, "平台不能为空")
	}
	exists, err := repo.R().ExistsClientVersion(version, platform, excludeId)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errs.NewError(400, "该平台下已存在相同版本号")
	}

	status := req.Status
	if status == "" {
		status = consts.StatusEnabled
	}

	return &model.ClientVersion{
		Version:     version,
		Platform:    platform,
		DownloadURL: strings.TrimSpace(req.DownloadURL),
		ForceUpdate: req.ForceUpdate,
		Status:      status,
		ReleaseNote: req.ReleaseNote,
	}, nil
}
