// model/vessel.go
package model

import "time"

type Vessel struct {
	VesselID          int64      `gorm:"primaryKey;autoIncrement;column:vessel_id" json:"vessel_id"`
	VesselName        string     `gorm:"column:vessel_name;not null" json:"vessel_name"`
	CallSign          *string    `gorm:"column:call_sign" json:"call_sign"`
	IMONumber         string     `gorm:"column:imo_number;not null;uniqueIndex:uk_imo_delete,where:delete_time IS NULL" json:"imo_number"`
	VesselType        *string    `gorm:"column:vessel_type" json:"vessel_type"`
	MaxDeadweightTon  *float64   `gorm:"column:max_deadweight_ton;type:decimal(12,2)" json:"max_deadweight_ton"`
	GrossTonnage      *float64   `gorm:"column:gross_tonnage;type:decimal(12,2)" json:"gross_tonnage"`
	NetTonnage        *float64   `gorm:"column:net_tonnage;type:decimal(12,2)" json:"net_tonnage"`
	DraftMeter        *float64   `gorm:"column:draft_meter;type:decimal(6,2)" json:"draft_meter"`
	SpeedKnot         *float64   `gorm:"column:speed_knot;type:decimal(6,2)" json:"speed_knot"`
	ContainerTEU      *int32     `gorm:"column:container_teu" json:"container_teu"`
	IsAvailable       int8       `gorm:"column:is_available;default:1" json:"is_available"`
	ShippingCompanyID *int64     `gorm:"column:shipping_company_id;index:idx_vessel_shipping_company_id" json:"shipping_company_id"`
	CreateTime        time.Time  `gorm:"column:create_time;autoCreateTime" json:"create_time"`
	UpdateTime        time.Time  `gorm:"column:update_time;autoUpdateTime" json:"update_time"`
	DeleteTime        *time.Time `gorm:"column:delete_time;index" json:"delete_time"`
	// Relationships
	ShippingCompany *ShippingCompany `gorm:"foreignKey:ShippingCompanyID;references:CompanyID" json:"shipping_company,omitempty"`
}

func (Vessel) TableName() string { return "vessel" }
