package dao

import (
	"backend/internal/model"

	"gorm.io/gorm"
)

type OrderCargoDAO interface {
	Create(cargo *model.OrderCargo) error
	GetByID(id int64) (*model.OrderCargo, error)
	Update(cargo *model.OrderCargo) error
	Delete(id int64) error
	ListByOrder(orderID int64) ([]model.OrderCargo, error)
	ListAll(page, pageSize int) ([]model.OrderCargo, int64, error)
}

type orderCargoDAOImpl struct {
	db *gorm.DB
}

func NewOrderCargoDAO(db *gorm.DB) OrderCargoDAO {
	return &orderCargoDAOImpl{db: db}
}

func (d *orderCargoDAOImpl) Create(cargo *model.OrderCargo) error {
	return d.db.Create(cargo).Error
}

func (d *orderCargoDAOImpl) GetByID(id int64) (*model.OrderCargo, error) {
	var cargo model.OrderCargo
	err := d.db.Scopes(NotDeleted).First(&cargo, id).Error
	return &cargo, err
}

func (d *orderCargoDAOImpl) Update(cargo *model.OrderCargo) error {
	return d.db.Save(cargo).Error
}

func (d *orderCargoDAOImpl) Delete(id int64) error {
	return d.db.Model(&model.OrderCargo{}).
		Where("detail_id = ?", id).
		Update("delete_time", gorm.Expr("NOW()")).Error
}

func (d *orderCargoDAOImpl) ListByOrder(orderID int64) ([]model.OrderCargo, error) {
	var cargos []model.OrderCargo
	err := d.db.Scopes(NotDeleted).Where("order_id = ?", orderID).Find(&cargos).Error
	return cargos, err
}

func (d *orderCargoDAOImpl) ListAll(page, pageSize int) ([]model.OrderCargo, int64, error) {
	var cargos []model.OrderCargo
	var total int64
	query := d.db.Model(&model.OrderCargo{}).Scopes(NotDeleted)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	err := query.Preload("Order").Offset(offset).Limit(pageSize).Find(&cargos).Error
	return cargos, total, err
}
