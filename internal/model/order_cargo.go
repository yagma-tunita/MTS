// model/order_cargo.go
package model

import "time"

type OrderCargo struct {
	DetailID         int64      `gorm:"primaryKey;autoIncrement;column:detail_id" json:"detail_id"`
	OrderID          *int64     `gorm:"column:order_id;index:idx_cargo_order_id" json:"order_id"`
	CargoName        *string    `gorm:"column:cargo_name" json:"cargo_name"`
	CargoType        *string    `gorm:"column:cargo_type" json:"cargo_type"`
	Quantity         *float64   `gorm:"column:quantity;type:decimal(18,2)" json:"quantity"`
	WeightTon        *float64   `gorm:"column:weight_ton;type:decimal(18,3)" json:"weight_ton"`
	VolumeCubicMeter *float64   `gorm:"column:volume_cubic_meter;type:decimal(18,3)" json:"volume_cubic_meter"`
	UnitPrice        *float64   `gorm:"column:unit_price;type:decimal(18,2)" json:"unit_price"`
	Subtotal         *float64   `gorm:"column:subtotal;type:decimal(18,2)" json:"subtotal"`
	CreateTime       time.Time  `gorm:"column:create_time;autoCreateTime" json:"create_time"`
	UpdateTime       time.Time  `gorm:"column:update_time;autoUpdateTime" json:"update_time"`
	DeleteTime       *time.Time `gorm:"column:delete_time;index" json:"delete_time"`
	// Relationships
	Order *ShippingOrder `gorm:"foreignKey:OrderID;references:OrderID" json:"order,omitempty"`
}

func (OrderCargo) TableName() string { return "order_cargo" }
