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
	List(page, pageSize int, keyword string) ([]model.Vessel, int64, error)
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
	err := d.db.Scopes(NotDeleted).Preload("ShippingCompany", func(db *gorm.DB) *gorm.DB {
		return db.Select("company_id", "company_name")
	}).First(&vessel, id).Error
	return &vessel, err
}

func (d *vesselDAOImpl) GetByIMONumber(imo string) (*model.Vessel, error) {
	var vessel model.Vessel
	err := d.db.Scopes(NotDeleted).Where("imo_number = ?", imo).First(&vessel).Error
	return &vessel, err
}

func (d *vesselDAOImpl) Update(vessel *model.Vessel) error {
	updates := map[string]interface{}{
		"vessel_name": vessel.VesselName,
		"call_sign":   vessel.CallSign,
		"imo_number":  vessel.IMONumber,
		"vessel_type": vessel.VesselType,
	}
	if vessel.MaxDeadweightTon != nil { updates["max_deadweight_ton"] = *vessel.MaxDeadweightTon }
	if vessel.GrossTonnage != nil { updates["gross_tonnage"] = *vessel.GrossTonnage }
	if vessel.NetTonnage != nil { updates["net_tonnage"] = *vessel.NetTonnage }
	if vessel.DraftMeter != nil { updates["draft_meter"] = *vessel.DraftMeter }
	if vessel.SpeedKnot != nil { updates["speed_knot"] = *vessel.SpeedKnot }
	if vessel.ContainerTEU != nil { updates["container_teu"] = *vessel.ContainerTEU }
	updates["is_available"] = vessel.IsAvailable
	if vessel.ShippingCompanyID != nil { updates["shipping_company_id"] = *vessel.ShippingCompanyID }
	return d.db.Model(&model.Vessel{}).Where("vessel_id = ?", vessel.VesselID).Omit("ShippingCompany").Updates(updates).Error
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
	err := query.Preload("ShippingCompany", func(db *gorm.DB) *gorm.DB {
		return db.Select("company_id", "company_name")
	}).Offset(offset).Limit(pageSize).Find(&vessels).Error
	return vessels, total, err
}

func (d *vesselDAOImpl) List(page, pageSize int, keyword string) ([]model.Vessel, int64, error) {
	var vessels []model.Vessel
	var total int64
	query := d.db.Model(&model.Vessel{}).Scopes(NotDeleted)
	if keyword != "" {
		query = query.Where("(vessel_name LIKE ? OR imo_number LIKE ?)", "%"+keyword+"%", "%"+keyword+"%")
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	err := query.Preload("ShippingCompany", func(db *gorm.DB) *gorm.DB {
		return db.Select("company_id", "company_name")
	}).Offset(offset).Limit(pageSize).Find(&vessels).Error
	return vessels, total, err
}
