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
	ListByShippingCompany(companyID int64, page, pageSize int, statusFilter *int8) ([]model.ShippingLine, int64, error)
	List(page, pageSize int, keyword string, statusFilter *int8) ([]model.ShippingLine, int64, error)
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
	err := d.db.Scopes(NotDeleted).Preload("ShippingCompany", func(db *gorm.DB) *gorm.DB {
		return db.Select("company_id", "company_name")
	}).First(&line, id).Error
	return &line, err
}

func (d *shippingLineDAOImpl) Update(line *model.ShippingLine) error {
	updates := map[string]interface{}{
		"line_name": line.LineName,
	}
	if line.ShippingCompanyID != nil { updates["shipping_company_id"] = *line.ShippingCompanyID }
	if line.PortSequence != nil { updates["port_sequence"] = *line.PortSequence }
	if line.TotalDistanceNm != nil { updates["total_distance_nm"] = *line.TotalDistanceNm }
	if line.DeparturePortName != nil { updates["departure_port_name"] = *line.DeparturePortName }
	if line.DestinationPortName != nil { updates["destination_port_name"] = *line.DestinationPortName }
	if line.Description != nil { updates["description"] = *line.Description }
	return d.db.Model(&model.ShippingLine{}).Where("line_id = ?", line.LineID).Omit("ShippingCompany").Updates(updates).Error
}

func (d *shippingLineDAOImpl) Delete(id int64) error {
	return d.db.Model(&model.ShippingLine{}).
		Where("line_id = ?", id).
		Update("delete_time", gorm.Expr("NOW()")).Error
}

func (d *shippingLineDAOImpl) ListByShippingCompany(companyID int64, page, pageSize int, statusFilter *int8) ([]model.ShippingLine, int64, error) {
	var lines []model.ShippingLine
	var total int64
	query := d.db.Model(&model.ShippingLine{}).Scopes(NotDeleted).Where("shipping_company_id = ?", companyID)
	if statusFilter != nil {
		query = query.Where("line_status = ?", *statusFilter)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	err := query.Preload("ShippingCompany", func(db *gorm.DB) *gorm.DB {
		return db.Select("company_id", "company_name")
	}).Offset(offset).Limit(pageSize).Find(&lines).Error
	return lines, total, err
}

func (d *shippingLineDAOImpl) List(page, pageSize int, keyword string, statusFilter *int8) ([]model.ShippingLine, int64, error) {
	var lines []model.ShippingLine
	var total int64
	query := d.db.Model(&model.ShippingLine{}).Scopes(NotDeleted)
	if keyword != "" {
		query = query.Where("line_name LIKE ?", "%"+keyword+"%")
	}
	if statusFilter != nil {
		query = query.Where("line_status = ?", *statusFilter)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	err := query.Preload("ShippingCompany", func(db *gorm.DB) *gorm.DB {
		return db.Select("company_id", "company_name")
	}).Offset(offset).Limit(pageSize).Find(&lines).Error
	return lines, total, err
}
