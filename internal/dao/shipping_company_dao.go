package dao

import (
	"backend/internal/model"

	"gorm.io/gorm"
)

type ShippingCompanyDAO interface {
	Create(company *model.ShippingCompany) error
	GetByID(id int64) (*model.ShippingCompany, error)
	GetByUsername(username string) (*model.ShippingCompany, error)
	Update(company *model.ShippingCompany) error
	Delete(id int64) error
	List(page, pageSize int) ([]model.ShippingCompany, int64, error)
}

type shippingCompanyDAOImpl struct {
	db *gorm.DB
}

func NewShippingCompanyDAO(db *gorm.DB) ShippingCompanyDAO {
	return &shippingCompanyDAOImpl{db: db}
}

func (d *shippingCompanyDAOImpl) Create(company *model.ShippingCompany) error {
	return d.db.Create(company).Error
}

func (d *shippingCompanyDAOImpl) GetByID(id int64) (*model.ShippingCompany, error) {
	var company model.ShippingCompany
	err := d.db.Scopes(NotDeleted).First(&company, id).Error
	return &company, err
}

func (d *shippingCompanyDAOImpl) GetByUsername(username string) (*model.ShippingCompany, error) {
	var company model.ShippingCompany
	err := d.db.Scopes(NotDeleted).
		Where("login_username = ?", username).
		First(&company).Error
	return &company, err
}

func (d *shippingCompanyDAOImpl) Update(company *model.ShippingCompany) error {
	return d.db.Save(company).Error
}

func (d *shippingCompanyDAOImpl) Delete(id int64) error {
	return d.db.Model(&model.ShippingCompany{}).
		Where("company_id = ?", id).
		Update("delete_time", gorm.Expr("NOW()")).Error
}

func (d *shippingCompanyDAOImpl) List(page, pageSize int) ([]model.ShippingCompany, int64, error) {
	var companies []model.ShippingCompany
	var total int64
	query := d.db.Model(&model.ShippingCompany{}).Scopes(NotDeleted)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Find(&companies).Error
	return companies, total, err
}
