package model

import (
	"database/sql/driver"
	"encoding/json"

	"server-cluster/share/enum"
)

// AccountModel 账号表模型。
type AccountModel struct {
	ID        int64  `gorm:"primaryKey;column:id" json:"id"`
	PlayerID  int64  `gorm:"column:playerId" json:"player_id"`
	Username  string `gorm:"column:username;size:64" json:"username"`
	Password  string `gorm:"column:password;size:256" json:"password"`
	CreatedAt int64  `gorm:"column:created_at" json:"created_at"`
}

// TableName 指定表名。
func (AccountModel) TableName() string { return enum.TableAccount.String() }

func (e *AccountModel) Marshal() ([]byte, error)    { return json.Marshal(e) }
func (e *AccountModel) Unmarshal(data []byte) error { return json.Unmarshal(data, e) }
func (e *AccountModel) Scan(src any) error          { return ScanJSON(src, e) }
func (e AccountModel) Value() (driver.Value, error) { return ValueJSON(e) }
