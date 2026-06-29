// model/berth.go
package model

import "time"

type Berth struct {
	BerthID            int64      `gorm:"primaryKey;autoIncrement;column:berth_id" json:"berth_id"`
	BerthName          string     `gorm:"column:berth_name;not null" json:"berth_name"`
	PortID             *int64     `gorm:"column:port_id;index:idx_berth_port_id" json:"port_id"`
	BerthType          *string    `gorm:"column:berth_type" json:"berth_type"`
	DraftMeter         *float64   `gorm:"column:draft_meter;type:decimal(6,2)" json:"draft_meter"`
	LengthMeter        *float64   `gorm:"column:length_meter;type:decimal(8,2)" json:"length_meter"`
	WidthMeter         *float64   `gorm:"column:width_meter;type:decimal(8,2)" json:"width_meter"`
	MaxBerthingTonnage *float64   `gorm:"column:max_berthing_tonnage;type:decimal(12,2)" json:"max_berthing_tonnage"`
	FunctionalZone     *string    `gorm:"column:functional_zone" json:"functional_zone"`
	IsAvailable        int8       `gorm:"column:is_available;default:1" json:"is_available"`
	CreateTime         time.Time  `gorm:"column:create_time;autoCreateTime" json:"create_time"`
	UpdateTime         time.Time  `gorm:"column:update_time;autoUpdateTime" json:"update_time"`
	DeleteTime         *time.Time `gorm:"column:delete_time;index" json:"delete_time"`
	// Relationships
	Port *Port `gorm:"foreignKey:PortID;references:PortID" json:"port,omitempty"`
}

func (Berth) TableName() string { return "berth" }
