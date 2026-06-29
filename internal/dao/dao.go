package dao

import (
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB(db *gorm.DB) {
	DB = db
}

func GetDB() *gorm.DB {
	return DB
}

func NotDeleted(db *gorm.DB) *gorm.DB {
	return db.Where("delete_time IS NULL")
}
