package model

import "time"

type LineVessel struct {
	LineID     int64     `gorm:"primaryKey;column:line_id"`
	VesselID   int64     `gorm:"primaryKey;column:vessel_id"`
	CreateTime time.Time `gorm:"column:create_time;autoCreateTime" json:"create_time"`
	Line   *ShippingLine `gorm:"foreignKey:LineID;references:LineID" json:"line,omitempty"`
	Vessel *Vessel       `gorm:"foreignKey:VesselID;references:VesselID" json:"vessel,omitempty"`
}

func (LineVessel) TableName() string { return "line_vessel" }
