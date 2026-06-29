// model/port.go
package model

import "time"

type Port struct {
	PortID        int64      `gorm:"primaryKey;autoIncrement;column:port_id" json:"port_id"`
	PortName      string     `gorm:"column:port_name;not null" json:"port_name"`
	PortCode      *string    `gorm:"column:port_code;unique" json:"port_code"`
	CityID        *int64     `gorm:"column:city_id;index:idx_port_city_id" json:"city_id"`
	Latitude      *float64   `gorm:"column:latitude;type:decimal(10,6)" json:"latitude"`
	Longitude     *float64   `gorm:"column:longitude;type:decimal(10,6)" json:"longitude"`
	PortType      *string    `gorm:"column:port_type" json:"port_type"`
	MaxDraftMeter *float64   `gorm:"column:max_draft_meter;type:decimal(6,2)" json:"max_draft_meter"`
	CreateTime    time.Time  `gorm:"column:create_time;autoCreateTime" json:"create_time"`
	UpdateTime    time.Time  `gorm:"column:update_time;autoUpdateTime" json:"update_time"`
	DeleteTime    *time.Time `gorm:"column:delete_time;index" json:"delete_time"`
	// Relationships
	City *City `gorm:"foreignKey:CityID;references:CityID" json:"city,omitempty"`
}

func (Port) TableName() string { return "port" }
