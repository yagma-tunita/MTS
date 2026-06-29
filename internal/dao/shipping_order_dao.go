package dao

import (
	"backend/internal/model"

	"gorm.io/gorm"
)

type ShippingOrderDAO interface {
	Create(order *model.ShippingOrder) error
	GetByID(id int64) (*model.ShippingOrder, error)
	GetByOrderNo(orderNo string) (*model.ShippingOrder, error)
	Update(order *model.ShippingOrder) error
	Delete(id int64) error
	ListByShipper(companyID int64, page, pageSize int) ([]model.ShippingOrder, int64, error)
	List(page, pageSize int) ([]model.ShippingOrder, int64, error)
}

type shippingOrderDAOImpl struct {
	db *gorm.DB
}

func NewShippingOrderDAO(db *gorm.DB) ShippingOrderDAO {
	return &shippingOrderDAOImpl{db: db}
}

func (d *shippingOrderDAOImpl) Create(order *model.ShippingOrder) error {
	return d.db.Create(order).Error
}

func (d *shippingOrderDAOImpl) GetByID(id int64) (*model.ShippingOrder, error) {
	var order model.ShippingOrder
	err := d.db.Scopes(NotDeleted).First(&order, id).Error
	return &order, err
}

func (d *shippingOrderDAOImpl) GetByOrderNo(orderNo string) (*model.ShippingOrder, error) {
	var order model.ShippingOrder
	err := d.db.Scopes(NotDeleted).Where("order_no = ?", orderNo).First(&order).Error
	return &order, err
}

func (d *shippingOrderDAOImpl) Update(order *model.ShippingOrder) error {
	return d.db.Save(order).Error
}

func (d *shippingOrderDAOImpl) Delete(id int64) error {
	return d.db.Model(&model.ShippingOrder{}).
		Where("order_id = ?", id).
		Update("delete_time", gorm.Expr("NOW()")).Error
}

func (d *shippingOrderDAOImpl) ListByShipper(companyID int64, page, pageSize int) ([]model.ShippingOrder, int64, error) {
	var orders []model.ShippingOrder
	var total int64
	query := d.db.Model(&model.ShippingOrder{}).Scopes(NotDeleted).Where("shipper_company_id = ?", companyID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Find(&orders).Error
	return orders, total, err
}

func (d *shippingOrderDAOImpl) List(page, pageSize int) ([]model.ShippingOrder, int64, error) {
	var orders []model.ShippingOrder
	var total int64
	query := d.db.Model(&model.ShippingOrder{}).Scopes(NotDeleted)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Find(&orders).Error
	return orders, total, err
}
