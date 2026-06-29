// model/shipping_line.go
package model

import (
	"time"
)

type ShippingLine struct {
	LineID              int64      `gorm:"primaryKey;autoIncrement;column:line_id" json:"line_id"`
	LineName            string     `gorm:"column:line_name;not null" json:"line_name"`
	ShippingCompanyID   *int64     `gorm:"column:shipping_company_id;index:idx_shipping_line_company_id" json:"shipping_company_id"`
	PortSequence        *string    `gorm:"column:port_sequence;type:json" json:"port_sequence"` // store as JSON string
	TotalDistanceNm     *float64   `gorm:"column:total_distance_nm;type:decimal(10,2)" json:"total_distance_nm"`
	DeparturePortName   *string    `gorm:"column:departure_port_name" json:"departure_port_name"`
	DestinationPortName *string    `gorm:"column:destination_port_name" json:"destination_port_name"`
	Description         *string    `gorm:"column:description;type:text" json:"description"`
	CreateTime          time.Time  `gorm:"column:create_time;autoCreateTime" json:"create_time"`
	UpdateTime          time.Time  `gorm:"column:update_time;autoUpdateTime" json:"update_time"`
	DeleteTime          *time.Time `gorm:"column:delete_time;index" json:"delete_time"`
	// Relationships
	ShippingCompany *ShippingCompany `gorm:"foreignKey:ShippingCompanyID;references:CompanyID" json:"shipping_company,omitempty"`
}

func (ShippingLine) TableName() string { return "shipping_line" }
