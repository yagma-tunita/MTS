// model/shipping_order.go
package model

import "time"

type ShippingOrder struct {
	OrderID               int64      `gorm:"primaryKey;autoIncrement;column:order_id" json:"order_id"`
	OrderNo               string     `gorm:"column:order_no;not null;uniqueIndex:uk_orderno_delete,where:delete_time IS NULL" json:"order_no"`
	ShipperCompanyID      *int64     `gorm:"column:shipper_company_id;index:idx_order_shipper_company_id" json:"shipper_company_id"`
	CityID                *int64     `gorm:"column:city_id;index:idx_order_city_id" json:"city_id"`
	LoadNoteID            *int64     `gorm:"column:load_note_id;index:idx_order_load_note_id" json:"load_note_id"`
	UnloadNoteID          *int64     `gorm:"column:unload_note_id;index:idx_order_unload_note_id" json:"unload_note_id"`
	DeparturePortID       *int64     `gorm:"column:departure_port_id;index:idx_order_departure_port_id" json:"departure_port_id"`
	DestinationPortID     *int64     `gorm:"column:destination_port_id;index:idx_order_destination_port_id" json:"destination_port_id"`
	ExpectedDepartureDate *time.Time `gorm:"column:expected_departure_date;type:date" json:"expected_departure_date"`
	ExpectedArrivalDate   *time.Time `gorm:"column:expected_arrival_date;type:date" json:"expected_arrival_date"`
	TotalCost             *float64   `gorm:"column:total_cost;type:decimal(18,2)" json:"total_cost"`
	ShipperContact        *string    `gorm:"column:shipper_contact" json:"shipper_contact"`
	ConsigneeContact      *string    `gorm:"column:consignee_contact" json:"consignee_contact"`
	PaymentStatus         *int8      `gorm:"column:payment_status" json:"payment_status"`
	OrderStatus           *int8      `gorm:"column:order_status" json:"order_status"`
	TotalWeightTon        *float64   `gorm:"column:total_weight_ton;type:decimal(18,3)" json:"total_weight_ton"`
	TotalVolumeCubicMeter *float64   `gorm:"column:total_volume_cubic_meter;type:decimal(18,3)" json:"total_volume_cubic_meter"`
	CreateTime            time.Time  `gorm:"column:create_time;autoCreateTime" json:"create_time"`
	UpdateTime            time.Time  `gorm:"column:update_time;autoUpdateTime" json:"update_time"`
	DeleteTime            *time.Time `gorm:"column:delete_time;index" json:"delete_time"`
	// Relationships
	ShipperCompany  *ShipperCompany  `gorm:"foreignKey:ShipperCompanyID;references:CompanyID" json:"shipper_company,omitempty"`
	City            *City            `gorm:"foreignKey:CityID;references:CityID" json:"city,omitempty"`
	LoadNote        *VoyageCargoNote `gorm:"foreignKey:LoadNoteID;references:NoteID" json:"load_note,omitempty"`
	UnloadNote      *VoyageCargoNote `gorm:"foreignKey:UnloadNoteID;references:NoteID" json:"unload_note,omitempty"`
	DeparturePort   *Port            `gorm:"foreignKey:DeparturePortID;references:PortID" json:"departure_port,omitempty"`
	DestinationPort *Port            `gorm:"foreignKey:DestinationPortID;references:PortID" json:"destination_port,omitempty"`
	OrderCargos     []OrderCargo     `gorm:"foreignKey:OrderID;references:OrderID" json:"order_cargos,omitempty"`
}

func (ShippingOrder) TableName() string { return "shipping_order" }
