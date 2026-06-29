package dao

import (
	"backend/internal/model"

	"gorm.io/gorm"
)

type ShipperCompanyDAO interface {
	Create(company *model.ShipperCompany) error
	GetByID(id int64) (*model.ShipperCompany, error)
	GetByUsername(username string) (*model.ShipperCompany, error)
	Update(company *model.ShipperCompany) error
	Delete(id int64) error
	List(page, pageSize int) ([]model.ShipperCompany, int64, error)
}

type shipperCompanyDAOImpl struct {
	db *gorm.DB
}

func NewShipperCompanyDAO(db *gorm.DB) ShipperCompanyDAO {
	return &shipperCompanyDAOImpl{db: db}
}

func (d *shipperCompanyDAOImpl) Create(company *model.ShipperCompany) error {
	return d.db.Create(company).Error
}

func (d *shipperCompanyDAOImpl) GetByID(id int64) (*model.ShipperCompany, error) {
	var company model.ShipperCompany
	err := d.db.Scopes(NotDeleted).First(&company, id).Error
	return &company, err
}

func (d *shipperCompanyDAOImpl) GetByUsername(username string) (*model.ShipperCompany, error) {
	var company model.ShipperCompany
	err := d.db.Scopes(NotDeleted).
		Where("login_username = ?", username).
		First(&company).Error
	return &company, err
}

func (d *shipperCompanyDAOImpl) Update(company *model.ShipperCompany) error {
	return d.db.Save(company).Error
}

func (d *shipperCompanyDAOImpl) Delete(id int64) error {
	return d.db.Model(&model.ShipperCompany{}).
		Where("company_id = ?", id).
		Update("delete_time", gorm.Expr("NOW()")).Error
}

func (d *shipperCompanyDAOImpl) List(page, pageSize int) ([]model.ShipperCompany, int64, error) {
	var companies []model.ShipperCompany
	var total int64
	query := d.db.Model(&model.ShipperCompany{}).Scopes(NotDeleted)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Find(&companies).Error
	return companies, total, err
}
