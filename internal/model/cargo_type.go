package model

import "time"

type CargoType struct {
	TypeID      int64      `gorm:"primaryKey;autoIncrement;column:type_id" json:"type_id"`
	TypeName    string     `gorm:"column:type_name;not null" json:"type_name"`
	TypeCode    string     `gorm:"column:type_code;not null;uniqueIndex" json:"type_code"`
	Description *string    `gorm:"column:description" json:"description"`
	CreateTime  time.Time  `gorm:"column:create_time;autoCreateTime" json:"create_time"`
	UpdateTime  time.Time  `gorm:"column:update_time;autoUpdateTime" json:"update_time"`
	DeleteTime  *time.Time `gorm:"column:delete_time;index" json:"delete_time"`
}

func (CargoType) TableName() string { return "cargo_type" }
