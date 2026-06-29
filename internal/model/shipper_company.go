// model/shipper_company.go
package model

import "time"

type ShipperCompany struct {
	CompanyID               int64      `gorm:"primaryKey;autoIncrement;column:company_id" json:"company_id"`
	CompanyName             string     `gorm:"column:company_name;not null" json:"company_name"`
	UnifiedSocialCreditCode *string    `gorm:"column:unified_social_credit_code;uniqueIndex:uk_social_credit_delete,where:delete_time IS NULL" json:"unified_social_credit_code"`
	LegalRepresentative     *string    `gorm:"column:legal_representative" json:"legal_representative"`
	ContactPhone            *string    `gorm:"column:contact_phone" json:"contact_phone"`
	Address                 *string    `gorm:"column:address" json:"address"`
	LoginUsername           string     `gorm:"column:login_username;not null;uniqueIndex:uk_username_delete,where:delete_time IS NULL" json:"login_username"`
	LoginPassword           string     `gorm:"column:login_password;not null" json:"login_password"`
	AccountStatus           int8       `gorm:"column:account_status;default:1" json:"account_status"`
	CreateTime              time.Time  `gorm:"column:create_time;autoCreateTime" json:"create_time"`
	UpdateTime              time.Time  `gorm:"column:update_time;autoUpdateTime" json:"update_time"`
	DeleteTime              *time.Time `gorm:"column:delete_time;index" json:"delete_time"`
}

func (ShipperCompany) TableName() string { return "shipper_company" }
