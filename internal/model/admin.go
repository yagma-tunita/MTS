// model/admin.go
package model

import "time"

type Admin struct {
	AdminID    int64      `gorm:"primaryKey;autoIncrement;column:admin_id" json:"admin_id"`
	Username   string     `gorm:"column:username;not null;uniqueIndex:uk_username_delete,where:delete_time IS NULL" json:"username"`
	Password   string     `gorm:"column:password;not null" json:"password"`
	RealName   *string    `gorm:"column:real_name" json:"real_name"`
	Role       int8       `gorm:"column:role;default:2" json:"role"`
	CreateTime time.Time  `gorm:"column:create_time;autoCreateTime" json:"create_time"`
	UpdateTime time.Time  `gorm:"column:update_time;autoUpdateTime" json:"update_time"`
	DeleteTime *time.Time `gorm:"column:delete_time;index" json:"delete_time"`
}

func (Admin) TableName() string { return "admin" }
