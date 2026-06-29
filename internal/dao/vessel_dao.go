package dao

import (
	"backend/internal/model"

	"gorm.io/gorm"
)

type VesselDAO interface {
	Create(vessel *model.Vessel) error
	GetByID(id int64) (*model.Vessel, error)
	GetByIMONumber(imo string) (*model.Vessel, error)
	Update(vessel *model.Vessel) error
	Delete(id int64) error
	ListByShippingCompany(companyID int64, page, pageSize int) ([]model.Vessel, int64, error)
	List(page, pageSize int) ([]model.Vessel, int64, error)
}

type vesselDAOImpl struct {
	db *gorm.DB
}

func NewVesselDAO(db *gorm.DB) VesselDAO {
	return &vesselDAOImpl{db: db}
}

func (d *vesselDAOImpl) Create(vessel *model.Vessel) error {
	return d.db.Create(vessel).Error
}

func (d *vesselDAOImpl) GetByID(id int64) (*model.Vessel, error) {
	var vessel model.Vessel
	err := d.db.Scopes(NotDeleted).First(&vessel, id).Error
	return &vessel, err
}

func (d *vesselDAOImpl) GetByIMONumber(imo string) (*model.Vessel, error) {
	var vessel model.Vessel
	err := d.db.Scopes(NotDeleted).Where("imo_number = ?", imo).First(&vessel).Error
	return &vessel, err
}

func (d *vesselDAOImpl) Update(vessel *model.Vessel) error {
	return d.db.Save(vessel).Error
}

func (d *vesselDAOImpl) Delete(id int64) error {
	return d.db.Model(&model.Vessel{}).
		Where("vessel_id = ?", id).
		Update("delete_time", gorm.Expr("NOW()")).Error
}

func (d *vesselDAOImpl) ListByShippingCompany(companyID int64, page, pageSize int) ([]model.Vessel, int64, error) {
	var vessels []model.Vessel
	var total int64
	query := d.db.Model(&model.Vessel{}).Scopes(NotDeleted).Where("shipping_company_id = ?", companyID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Find(&vessels).Error
	return vessels, total, err
}

func (d *vesselDAOImpl) List(page, pageSize int) ([]model.Vessel, int64, error) {
	var vessels []model.Vessel
	var total int64
	query := d.db.Model(&model.Vessel{}).Scopes(NotDeleted)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Find(&vessels).Error
	return vessels, total, err
}
