package dao

import (
	"gorm.io/gorm"
)

func NotDeleted(db *gorm.DB) *gorm.DB {
	table := db.Statement.Table
	if table == "" && db.Statement.Model != nil {
		db.Statement.Parse(db.Statement.Model)
		table = db.Statement.Table
	}
	if table != "" {
		return db.Where(table + ".delete_time IS NULL")
	}
	return db.Where("delete_time IS NULL")
}
