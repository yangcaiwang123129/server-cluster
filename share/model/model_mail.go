package model

import (
	"database/sql/driver"
	"encoding/json"
	"server-cluster/share/enum"
	"server-cluster/share/util"
)

// 邮件状态：未读 → 已读 → 已领取。
const (
	MailStatusUnread  int32 = 1 // 未读
	MailStatusRead    int32 = 2 // 已读
	MailStatusClaimed int32 = 3 // 已领取（附件已发放，可删除）
)

// MailAttachment 邮件附件资产。
type MailAttachment struct {
	Type  int32 `json:"type"`   // 资产类型（对应 tpl.AssetType）
	CfgID int32 `json:"cfg_id"` // 配置ID
	Count int32 `json:"count"`  // 数量
}

// MailInfo 单封邮件的持有数据（player_mail.mails 的 map 元素，key=邮件实例ID）。
// 状态流转：未读 → 已读 → 已领取；已领取或无附件的邮件可删除。
type MailInfo struct {
	MailID      int64            `json:"mail_id"`     // 邮件实例ID（雪花）
	Title       string           `json:"title"`       // 标题
	Content     string           `json:"content"`     // 正文
	Attachments []MailAttachment `json:"attachments"` // 附件列表（空则为纯通知）
	Status      int32            `json:"status"`      // 状态（MailStatus*）
	CreatedAt   int64            `json:"created_at"`  // 发送时间（毫秒时间戳）
	ReadAt      int64            `json:"read_at"`     // 已读时间（0 未读）
	ClaimedAt   int64            `json:"claimed_at"`  // 领取时间（0 未领取）
}

// HasAttachments 是否携带附件。
func (m MailInfo) HasAttachments() bool { return len(m.Attachments) > 0 }

// CanDelete 是否可删除：无附件的邮件任意状态可删；带附件的邮件需先领取。
func (m MailInfo) CanDelete() bool {
	if !m.HasAttachments() {
		return true
	}
	return m.Status == MailStatusClaimed
}

// PlayerMail 玩家邮件持有（1:1 行，多邮件以 JSON 对象存储，key=邮件实例ID），对应 player_mail 表。
type PlayerMail struct {
	TableModel `gorm:"embedded"`
	PlayerID   int64                          `gorm:"column:player_id;type:bigint;not null;primaryKey;comment:玩家ID" json:"player_id"`
	Mails      *util.SafeMap[int64, MailInfo] `gorm:"column:mails;type:json;comment:邮件列表(JSON对象,key=邮件实例ID)" json:"mails"`
}

func (PlayerMail) TableName() string { return enum.TablePlayerMail.String() }

// AddMail 新增邮件；实例ID已存在返回 false。
func (p *PlayerMail) AddMail(m MailInfo) bool {
	_, ok := p.Mails.Update(m.MailID, func(_ MailInfo, exists bool) (MailInfo, bool) {
		if exists {
			return MailInfo{}, false
		}
		return m, true
	})
	return ok
}

// UpdateMail 覆盖邮件状态；不存在返回 false。
func (p *PlayerMail) UpdateMail(m MailInfo) bool {
	_, ok := p.Mails.Update(m.MailID, func(_ MailInfo, exists bool) (MailInfo, bool) {
		if !exists {
			return MailInfo{}, false
		}
		return m, true
	})
	return ok
}

// DeleteMail 删除邮件；不存在返回 false。
func (p *PlayerMail) DeleteMail(mailID int64) bool { return p.Mails.Delete(mailID) }

// ScanJSON/ValueJSON 由 model 包提供，这里实现 Serializable 以兼容 cache 序列化。
func (this *PlayerMail) Marshal() ([]byte, error)    { return json.Marshal(this) }
func (this *PlayerMail) Unmarshal(data []byte) error { return json.Unmarshal(data, this) }
func (this *PlayerMail) Scan(src any) error          { return ScanJSON(src, this) }
func (this PlayerMail) Value() (driver.Value, error) { return ValueJSON(this) }
