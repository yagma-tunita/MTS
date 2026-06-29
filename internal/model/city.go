// model/city.go
package model

import "time"

type City struct {
	CityID      int64      `gorm:"primaryKey;autoIncrement;column:city_id" json:"city_id"`
	CityName    string     `gorm:"column:city_name;not null" json:"city_name"`
	Country     *string    `gorm:"column:country" json:"country"`
	CountryCode *string    `gorm:"column:country_code" json:"country_code"`
	Timezone    *string    `gorm:"column:timezone" json:"timezone"`
	Latitude    *float64   `gorm:"column:latitude;type:decimal(10,6)" json:"latitude"`
	Longitude   *float64   `gorm:"column:longitude;type:decimal(10,6)" json:"longitude"`
	CreateTime  time.Time  `gorm:"column:create_time;autoCreateTime" json:"create_time"`
	UpdateTime  time.Time  `gorm:"column:update_time;autoUpdateTime" json:"update_time"`
	DeleteTime  *time.Time `gorm:"column:delete_time;index" json:"delete_time"`
}

func (City) TableName() string { return "city" }
