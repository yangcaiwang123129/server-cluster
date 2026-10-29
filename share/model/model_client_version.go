package model

import (
	"time"

	"server-cluster/share/enum"
)

// ClientVersion 客户端版本：后台发布的 U3D 安装包清单，客户端启动时据此判断是否需要更新。
type ClientVersion struct {
	VersionId   int       `json:"versionId" gorm:"column:version_id;type:int;not null;primaryKey;autoIncrement;comment:版本ID"`
	Version     string    `json:"version" gorm:"column:version;type:varchar(32);not null;default:'';index;comment:版本号"`
	Platform    string    `json:"platform" gorm:"column:platform;type:varchar(16);not null;default:'';comment:平台(android/ios/windows)"`
	DownloadURL string    `json:"downloadUrl" gorm:"column:download_url;type:varchar(255);not null;default:'';comment:下载地址"`
	ForceUpdate bool      `json:"forceUpdate" gorm:"column:force_update;type:tinyint(1);not null;default:0;comment:是否强制更新"`
	Status      string    `json:"status" gorm:"column:status;type:varchar(4);not null;default:2;comment:状态(1停用 2正常)"`
	ReleaseNote string    `json:"releaseNote" gorm:"column:release_note;type:varchar(512);not null;default:'';comment:更新说明"`
	CreatedAt   time.Time `json:"createdAt" gorm:"column:created_at;type:datetime;not null;comment:创建时间"`
	UpdatedAt   time.Time `json:"updatedAt" gorm:"column:updated_at;type:datetime;not null;comment:更新时间"`
}

// TableName 表名。
func (ClientVersion) TableName() string { return enum.TableClientVersion.String() }
