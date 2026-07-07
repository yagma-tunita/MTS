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
	List(page, pageSize int, cityName string) ([]model.City, int64, error)
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
	updates := map[string]interface{}{}
	if city.CityName != "" { updates["city_name"] = city.CityName }
	if city.Country != nil { updates["country"] = *city.Country }
	if city.CountryCode != nil { updates["country_code"] = *city.CountryCode }
	if city.Timezone != nil { updates["timezone"] = *city.Timezone }
	return d.db.Model(&model.City{}).Where("city_id = ?", city.CityID).Updates(updates).Error
}

func (d *cityDAOImpl) Delete(id int64) error {
	return d.db.Model(&model.City{}).
		Where("city_id = ?", id).
		Update("delete_time", gorm.Expr("NOW()")).Error
}

func (d *cityDAOImpl) List(page, pageSize int, cityName string) ([]model.City, int64, error) {
	var cities []model.City
	var total int64
	query := d.db.Model(&model.City{}).Scopes(NotDeleted)
	if cityName != "" {
		query = query.Where("city_name LIKE ?", "%"+cityName+"%")
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	err := query.Order("city_name ASC").Offset(offset).Limit(pageSize).Find(&cities).Error
	return cities, total, err
}
