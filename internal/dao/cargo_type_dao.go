package dao

import (
	"backend/internal/model"

	"gorm.io/gorm"
)

type CargoTypeDAO interface {
	Create(t *model.CargoType) error
	GetByID(id int64) (*model.CargoType, error)
	Update(t *model.CargoType) error
	Delete(id int64) error
	List(page, pageSize int, keyword string) ([]model.CargoType, int64, error)
	ListAll() ([]model.CargoType, error)
}

type cargoTypeDAOImpl struct {
	db *gorm.DB
}

func NewCargoTypeDAO(db *gorm.DB) CargoTypeDAO {
	return &cargoTypeDAOImpl{db: db}
}

func (d *cargoTypeDAOImpl) Create(t *model.CargoType) error {
	return d.db.Create(t).Error
}

func (d *cargoTypeDAOImpl) GetByID(id int64) (*model.CargoType, error) {
	var t model.CargoType
	err := d.db.Scopes(NotDeleted).First(&t, id).Error
	return &t, err
}

func (d *cargoTypeDAOImpl) Update(t *model.CargoType) error {
	updates := map[string]interface{}{
		"type_name": t.TypeName,
		"type_code": t.TypeCode,
		"description": t.Description,
	}
	return d.db.Model(&model.CargoType{}).Where("type_id = ?", t.TypeID).Updates(updates).Error
}

func (d *cargoTypeDAOImpl) Delete(id int64) error {
	return d.db.Model(&model.CargoType{}).
		Where("type_id = ?", id).
		Update("delete_time", gorm.Expr("NOW()")).Error
}

func (d *cargoTypeDAOImpl) List(page, pageSize int, keyword string) ([]model.CargoType, int64, error) {
	var types []model.CargoType
	var total int64
	query := d.db.Model(&model.CargoType{}).Scopes(NotDeleted)
	if keyword != "" {
		query = query.Where("type_name LIKE ? OR type_code LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Find(&types).Error
	return types, total, err
}

func (d *cargoTypeDAOImpl) ListAll() ([]model.CargoType, error) {
	var types []model.CargoType
	err := d.db.Scopes(NotDeleted).Find(&types).Error
	return types, err
}
