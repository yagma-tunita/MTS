package dao

import (
	"backend/internal/model"

	"gorm.io/gorm"
)

type CityDAO interface {
	Create(city *model.City) error
	GetByID(id int64) (*model.City, error)
	Update(city *model.City) error
	Delete(id int64) error
	List(page, pageSize int) ([]model.City, int64, error)
}

type cityDAOImpl struct {
	db *gorm.DB
}

func NewCityDAO(db *gorm.DB) CityDAO {
	return &cityDAOImpl{db: db}
}

func (d *cityDAOImpl) Create(city *model.City) error {
	return d.db.Create(city).Error
}

func (d *cityDAOImpl) GetByID(id int64) (*model.City, error) {
	var city model.City
	err := d.db.Scopes(NotDeleted).First(&city, id).Error
	return &city, err
}

func (d *cityDAOImpl) Update(city *model.City) error {
	return d.db.Save(city).Error
}

func (d *cityDAOImpl) Delete(id int64) error {
	return d.db.Model(&model.City{}).
		Where("city_id = ?", id).
		Update("delete_time", gorm.Expr("NOW()")).Error
}

func (d *cityDAOImpl) List(page, pageSize int) ([]model.City, int64, error) {
	var cities []model.City
	var total int64
	query := d.db.Model(&model.City{}).Scopes(NotDeleted)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Find(&cities).Error
	return cities, total, err
}
