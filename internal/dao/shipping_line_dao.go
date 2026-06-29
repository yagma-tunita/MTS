package dao

import (
	"backend/internal/model"

	"gorm.io/gorm"
)

type ShippingLineDAO interface {
	Create(line *model.ShippingLine) error
	GetByID(id int64) (*model.ShippingLine, error)
	Update(line *model.ShippingLine) error
	Delete(id int64) error
	ListByShippingCompany(companyID int64, page, pageSize int) ([]model.ShippingLine, int64, error)
	List(page, pageSize int) ([]model.ShippingLine, int64, error)
}

type shippingLineDAOImpl struct {
	db *gorm.DB
}

func NewShippingLineDAO(db *gorm.DB) ShippingLineDAO {
	return &shippingLineDAOImpl{db: db}
}

func (d *shippingLineDAOImpl) Create(line *model.ShippingLine) error {
	return d.db.Create(line).Error
}

func (d *shippingLineDAOImpl) GetByID(id int64) (*model.ShippingLine, error) {
	var line model.ShippingLine
	err := d.db.Scopes(NotDeleted).First(&line, id).Error
	return &line, err
}

func (d *shippingLineDAOImpl) Update(line *model.ShippingLine) error {
	return d.db.Save(line).Error
}

func (d *shippingLineDAOImpl) Delete(id int64) error {
	return d.db.Model(&model.ShippingLine{}).
		Where("line_id = ?", id).
		Update("delete_time", gorm.Expr("NOW()")).Error
}

func (d *shippingLineDAOImpl) ListByShippingCompany(companyID int64, page, pageSize int) ([]model.ShippingLine, int64, error) {
	var lines []model.ShippingLine
	var total int64
	query := d.db.Model(&model.ShippingLine{}).Scopes(NotDeleted).Where("shipping_company_id = ?", companyID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Find(&lines).Error
	return lines, total, err
}

func (d *shippingLineDAOImpl) List(page, pageSize int) ([]model.ShippingLine, int64, error) {
	var lines []model.ShippingLine
	var total int64
	query := d.db.Model(&model.ShippingLine{}).Scopes(NotDeleted)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Find(&lines).Error
	return lines, total, err
}
