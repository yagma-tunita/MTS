package dao

import (
	"backend/internal/model"

	"gorm.io/gorm"
)

type LineVesselDAO interface {
	Assign(lineID, vesselID int64) error
	Unassign(lineID, vesselID int64) error
	GetVesselIDsByLine(lineID int64) ([]int64, error)
	GetLinesByVessel(vesselID int64) ([]int64, error)
}

type lineVesselDAOImpl struct {
	db *gorm.DB
}

func NewLineVesselDAO(db *gorm.DB) LineVesselDAO {
	return &lineVesselDAOImpl{db: db}
}

func (d *lineVesselDAOImpl) Assign(lineID, vesselID int64) error {
	return d.db.Create(&model.LineVessel{LineID: lineID, VesselID: vesselID}).Error
}

func (d *lineVesselDAOImpl) Unassign(lineID, vesselID int64) error {
	return d.db.Delete(&model.LineVessel{}, "line_id = ? AND vessel_id = ?", lineID, vesselID).Error
}

func (d *lineVesselDAOImpl) GetVesselIDsByLine(lineID int64) ([]int64, error) {
	var ids []int64
	err := d.db.Model(&model.LineVessel{}).Where("line_id = ?", lineID).Pluck("vessel_id", &ids).Error
	return ids, err
}

func (d *lineVesselDAOImpl) GetLinesByVessel(vesselID int64) ([]int64, error) {
	var ids []int64
	err := d.db.Model(&model.LineVessel{}).Where("vessel_id = ?", vesselID).Pluck("line_id", &ids).Error
	return ids, err
}
