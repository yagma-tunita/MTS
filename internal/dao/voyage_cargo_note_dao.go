package dao

import (
	"backend/internal/model"

	"gorm.io/gorm"
)

type VoyageCargoNoteDAO interface {
	Create(note *model.VoyageCargoNote) error
	GetByID(id int64) (*model.VoyageCargoNote, error)
	Update(note *model.VoyageCargoNote) error
	Delete(id int64) error
	ListByVoyage(lineID, vesselID int64, voyageDate string) ([]model.VoyageCargoNote, error)
	List(page, pageSize int) ([]model.VoyageCargoNote, int64, error)

	// Additional methods
	FindByPortAndOp(lineID, vesselID int64, voyageDate string, portID int64, opType string) (*model.VoyageCargoNote, error)
	AddCumulativeCapacity(tx *gorm.DB, noteID int64, deltaTon float64) error
}

type voyageCargoNoteDAOImpl struct {
	db *gorm.DB
}

func NewVoyageCargoNoteDAO(db *gorm.DB) VoyageCargoNoteDAO {
	return &voyageCargoNoteDAOImpl{db: db}
}

func (d *voyageCargoNoteDAOImpl) Create(note *model.VoyageCargoNote) error {
	return d.db.Create(note).Error
}

func (d *voyageCargoNoteDAOImpl) GetByID(id int64) (*model.VoyageCargoNote, error) {
	var note model.VoyageCargoNote
	err := d.db.First(&note, id).Error
	return &note, err
}

func (d *voyageCargoNoteDAOImpl) Update(note *model.VoyageCargoNote) error {
	return d.db.Save(note).Error
}

func (d *voyageCargoNoteDAOImpl) Delete(id int64) error {
	return d.db.Delete(&model.VoyageCargoNote{}, id).Error
}

func (d *voyageCargoNoteDAOImpl) ListByVoyage(lineID, vesselID int64, voyageDate string) ([]model.VoyageCargoNote, error) {
	var notes []model.VoyageCargoNote
	err := d.db.Where("line_id = ? AND vessel_id = ? AND voyage_date = ?", lineID, vesselID, voyageDate).
		Order("sequence_no ASC").
		Find(&notes).Error
	return notes, err
}

func (d *voyageCargoNoteDAOImpl) List(page, pageSize int) ([]model.VoyageCargoNote, int64, error) {
	var notes []model.VoyageCargoNote
	var total int64
	query := d.db.Model(&model.VoyageCargoNote{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Find(&notes).Error
	return notes, total, err
}

// FindByPortAndOp finds a cargo note for given port and operation type (LOAD/UNLOAD).
// It joins voyage_berthing to match the port_id by sequence_no.
func (d *voyageCargoNoteDAOImpl) FindByPortAndOp(lineID, vesselID int64, voyageDate string, portID int64, opType string) (*model.VoyageCargoNote, error) {
	var note model.VoyageCargoNote
	err := d.db.
		Table("voyage_cargo_note").
		Select("voyage_cargo_note.*").
		Joins("JOIN voyage_berthing ON voyage_berthing.line_id = voyage_cargo_note.line_id AND voyage_berthing.vessel_id = voyage_cargo_note.vessel_id AND voyage_berthing.voyage_date = voyage_cargo_note.voyage_date AND voyage_berthing.sequence_no = voyage_cargo_note.sequence_no").
		Where("voyage_cargo_note.line_id = ? AND voyage_cargo_note.vessel_id = ? AND voyage_cargo_note.voyage_date = ? AND voyage_berthing.port_id = ? AND voyage_cargo_note.operation_type = ?",
			lineID, vesselID, voyageDate, portID, opType).
		First(&note).Error
	if err != nil {
		return nil, err
	}
	return &note, nil
}

// AddCumulativeCapacity updates cumulative_booked_capacity_ton by delta (can be negative).
// It accepts a transaction object to be used within a transaction.
func (d *voyageCargoNoteDAOImpl) AddCumulativeCapacity(tx *gorm.DB, noteID int64, deltaTon float64) error {
	return tx.Model(&model.VoyageCargoNote{}).
		Where("note_id = ?", noteID).
		Update("cumulative_booked_capacity_ton", gorm.Expr("cumulative_booked_capacity_ton + ?", deltaTon)).Error
}
