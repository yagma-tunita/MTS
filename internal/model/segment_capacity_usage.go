// model/segment_capacity_usage.go
package model

import "time"

type SegmentCapacityUsage struct {
	UsageID     int64     `gorm:"primaryKey;autoIncrement;column:usage_id" json:"usage_id"`
	OrderID     *int64    `gorm:"column:order_id;index:idx_usage_order_id" json:"order_id"`
	LineID      *int64    `gorm:"column:line_id;index:idx_usage_line_id" json:"line_id"`
	VesselID    *int64    `gorm:"column:vessel_id;index:idx_usage_vessel_id" json:"vessel_id"`
	VoyageDate  time.Time `gorm:"column:voyage_date;type:date;not null" json:"voyage_date"`
	StartPortID *int64    `gorm:"column:start_port_id;index:idx_usage_start_port_id" json:"start_port_id"`
	EndPortID   *int64    `gorm:"column:end_port_id;index:idx_usage_end_port_id" json:"end_port_id"`
	OccupiedTon float64   `gorm:"column:occupied_ton;type:decimal(18,3);not null" json:"occupied_ton"`
	CreateTime  time.Time `gorm:"column:create_time;autoCreateTime" json:"create_time"`
	// Relationships
	Order     *ShippingOrder `gorm:"foreignKey:OrderID;references:OrderID" json:"order,omitempty"`
	Line      *ShippingLine  `gorm:"foreignKey:LineID;references:LineID" json:"line,omitempty"`
	Vessel    *Vessel        `gorm:"foreignKey:VesselID;references:VesselID" json:"vessel,omitempty"`
	StartPort *Port          `gorm:"foreignKey:StartPortID;references:PortID" json:"start_port,omitempty"`
	EndPort   *Port          `gorm:"foreignKey:EndPortID;references:PortID" json:"end_port,omitempty"`
}

func (SegmentCapacityUsage) TableName() string { return "segment_capacity_usage" }
