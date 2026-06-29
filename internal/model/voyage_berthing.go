// model/voyage_berthing.go
package model

import "time"

type VoyageBerthing struct {
	BerthingID           int64      `gorm:"primaryKey;autoIncrement;column:berthing_id" json:"berthing_id"`
	LineID               *int64     `gorm:"column:line_id;index:idx_berthing_line_id" json:"line_id"`
	VesselID             *int64     `gorm:"column:vessel_id;index:idx_berthing_vessel_id" json:"vessel_id"`
	VoyageDate           time.Time  `gorm:"column:voyage_date;type:date;not null" json:"voyage_date"`
	SequenceNo           int32      `gorm:"column:sequence_no;not null" json:"sequence_no"`
	PortID               *int64     `gorm:"column:port_id;index:idx_berthing_port_id" json:"port_id"`
	BerthID              *int64     `gorm:"column:berth_id;index:idx_berthing_berth_id" json:"berth_id"`
	PlannedArrivalTime   *time.Time `gorm:"column:planned_arrival_time" json:"planned_arrival_time"`
	PlannedDepartureTime *time.Time `gorm:"column:planned_departure_time" json:"planned_departure_time"`
	ActualArrivalTime    *time.Time `gorm:"column:actual_arrival_time" json:"actual_arrival_time"`
	ActualDepartureTime  *time.Time `gorm:"column:actual_departure_time" json:"actual_departure_time"`
	DraftAtBerthingMeter *float64   `gorm:"column:draft_at_berthing_meter;type:decimal(6,2)" json:"draft_at_berthing_meter"`
	IsAdjustable         int8       `gorm:"column:is_adjustable;default:1" json:"is_adjustable"`
	CreateTime           time.Time  `gorm:"column:create_time;autoCreateTime" json:"create_time"`
	UpdateTime           time.Time  `gorm:"column:update_time;autoUpdateTime" json:"update_time"`
	// Relationships
	Line   *ShippingLine `gorm:"foreignKey:LineID;references:LineID" json:"line,omitempty"`
	Vessel *Vessel       `gorm:"foreignKey:VesselID;references:VesselID" json:"vessel,omitempty"`
	Port   *Port         `gorm:"foreignKey:PortID;references:PortID" json:"port,omitempty"`
	Berth  *Berth        `gorm:"foreignKey:BerthID;references:BerthID" json:"berth,omitempty"`
}

func (VoyageBerthing) TableName() string { return "voyage_berthing" }
