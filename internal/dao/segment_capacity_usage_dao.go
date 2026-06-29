package dao

import (
	"backend/internal/model"

	"gorm.io/gorm"
)

type SegmentCapacityUsageDAO interface {
	Create(usage *model.SegmentCapacityUsage) error
	GetByID(id int64) (*model.SegmentCapacityUsage, error)
	Delete(id int64) error
	GetOccupiedTons(lineID, vesselID int64, voyageDate string, startPortID, endPortID int64) (float64, error)
	ListByOrder(orderID int64) ([]model.SegmentCapacityUsage, error)
}

type segmentCapacityUsageDAOImpl struct {
	db *gorm.DB
}

func NewSegmentCapacityUsageDAO(db *gorm.DB) SegmentCapacityUsageDAO {
	return &segmentCapacityUsageDAOImpl{db: db}
}

func (d *segmentCapacityUsageDAOImpl) Create(usage *model.SegmentCapacityUsage) error {
	return d.db.Create(usage).Error
}

func (d *segmentCapacityUsageDAOImpl) GetByID(id int64) (*model.SegmentCapacityUsage, error) {
	var usage model.SegmentCapacityUsage
	err := d.db.First(&usage, id).Error
	return &usage, err
}

func (d *segmentCapacityUsageDAOImpl) Delete(id int64) error {
	return d.db.Delete(&model.SegmentCapacityUsage{}, id).Error
}

func (d *segmentCapacityUsageDAOImpl) GetOccupiedTons(lineID, vesselID int64, voyageDate string, startPortID, endPortID int64) (float64, error) {
	var sum float64
	err := d.db.Model(&model.SegmentCapacityUsage{}).
		Where("line_id = ? AND vessel_id = ? AND voyage_date = ? AND start_port_id = ? AND end_port_id = ?",
			lineID, vesselID, voyageDate, startPortID, endPortID).
		Select("COALESCE(SUM(occupied_ton), 0)").
		Scan(&sum).Error
	return sum, err
}

func (d *segmentCapacityUsageDAOImpl) ListByOrder(orderID int64) ([]model.SegmentCapacityUsage, error) {
	var usages []model.SegmentCapacityUsage
	err := d.db.Where("order_id = ?", orderID).Find(&usages).Error
	return usages, err
}
