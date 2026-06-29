// model/voyage_cargo_note.go
package model

import "time"

type VoyageCargoNote struct {
	NoteID                      int64     `gorm:"primaryKey;autoIncrement;column:note_id" json:"note_id"`
	LineID                      *int64    `gorm:"column:line_id;index:idx_cargonote_line_id" json:"line_id"`
	VesselID                    *int64    `gorm:"column:vessel_id;index:idx_cargonote_vessel_id" json:"vessel_id"`
	VoyageDate                  time.Time `gorm:"column:voyage_date;type:date;not null" json:"voyage_date"`
	SequenceNo                  int32     `gorm:"column:sequence_no;not null" json:"sequence_no"`
	CargoName                   *string   `gorm:"column:cargo_name" json:"cargo_name"`
	CargoType                   *string   `gorm:"column:cargo_type" json:"cargo_type"`
	Quantity                    *float64  `gorm:"column:quantity;type:decimal(18,2)" json:"quantity"`
	WeightTon                   *float64  `gorm:"column:weight_ton;type:decimal(18,3)" json:"weight_ton"`
	VolumeCubicMeter            *float64  `gorm:"column:volume_cubic_meter;type:decimal(18,3)" json:"volume_cubic_meter"`
	UnitPrice                   *float64  `gorm:"column:unit_price;type:decimal(18,2)" json:"unit_price"`
	Subtotal                    *float64  `gorm:"column:subtotal;type:decimal(18,2)" json:"subtotal"`
	OperationType               *string   `gorm:"column:operation_type" json:"operation_type"`
	CargoHandledTon             *float64  `gorm:"column:cargo_handled_ton;type:decimal(18,3)" json:"cargo_handled_ton"`
	CumulativeBookedCapacityTon *float64  `gorm:"column:cumulative_booked_capacity_ton;type:decimal(18,3)" json:"cumulative_booked_capacity_ton"`
	CreateTime                  time.Time `gorm:"column:create_time;autoCreateTime" json:"create_time"`
	UpdateTime                  time.Time `gorm:"column:update_time;autoUpdateTime" json:"update_time"`
	// Relationships
	Line   *ShippingLine `gorm:"foreignKey:LineID;references:LineID" json:"line,omitempty"`
	Vessel *Vessel       `gorm:"foreignKey:VesselID;references:VesselID" json:"vessel,omitempty"`
}

func (VoyageCargoNote) TableName() string { return "voyage_cargo_note" }
