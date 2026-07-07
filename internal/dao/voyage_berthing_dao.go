package dao

import (
	"backend/internal/model"

	"gorm.io/gorm"
)

type VoyageBerthingDAO interface {
	Create(berthing *model.VoyageBerthing) error
	GetByID(id int64) (*model.VoyageBerthing, error)
	Update(berthing *model.VoyageBerthing) error
	Delete(id int64) error
	ListByVoyage(lineID, vesselID int64, voyageDate string) ([]model.VoyageBerthing, error)
	List(page, pageSize int) ([]model.VoyageBerthing, int64, error)
	ListByShippingCompany(companyID int64) ([]model.VoyageBerthing, error)
}

type voyageBerthingDAOImpl struct {
	db *gorm.DB
}

func NewVoyageBerthingDAO(db *gorm.DB) VoyageBerthingDAO {
	return &voyageBerthingDAOImpl{db: db}
}

func (d *voyageBerthingDAOImpl) Create(berthing *model.VoyageBerthing) error {
	return d.db.Create(berthing).Error
}

func (d *voyageBerthingDAOImpl) GetByID(id int64) (*model.VoyageBerthing, error) {
	var berthing model.VoyageBerthing
	err := d.db.First(&berthing, id).Error
	return &berthing, err
}

func (d *voyageBerthingDAOImpl) Update(berthing *model.VoyageBerthing) error {
	return d.db.Save(berthing).Error
}

func (d *voyageBerthingDAOImpl) Delete(id int64) error {
	return d.db.Delete(&model.VoyageBerthing{}, id).Error
}

func (d *voyageBerthingDAOImpl) ListByVoyage(lineID, vesselID int64, voyageDate string) ([]model.VoyageBerthing, error) {
	var berthings []model.VoyageBerthing
	err := d.db.Where("line_id = ? AND vessel_id = ? AND voyage_date = ?", lineID, vesselID, voyageDate).
		Order("sequence_no ASC").
		Find(&berthings).Error
	return berthings, err
}

func (d *voyageBerthingDAOImpl) List(page, pageSize int) ([]model.VoyageBerthing, int64, error) {
	var berthings []model.VoyageBerthing
	var total int64
	query := d.db.Model(&model.VoyageBerthing{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Find(&berthings).Error
	return berthings, total, err
}

func (d *voyageBerthingDAOImpl) ListByShippingCompany(companyID int64) ([]model.VoyageBerthing, error) {
	var berthings []model.VoyageBerthing
	err := d.db.Table("voyage_berthing").
		Joins("JOIN shipping_line ON voyage_berthing.line_id = shipping_line.line_id").
		Where("shipping_line.shipping_company_id = ? AND shipping_line.delete_time IS NULL", companyID).
		Order("voyage_berthing.voyage_date DESC, voyage_berthing.sequence_no ASC").
		Preload("Port").
		Find(&berthings).Error
	return berthings, err
}
