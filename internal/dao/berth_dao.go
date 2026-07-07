package dao

import (
	"backend/internal/model"

	"gorm.io/gorm"
)

type BerthDAO interface {
	Create(berth *model.Berth) error
	GetByID(id int64) (*model.Berth, error)
	Update(berth *model.Berth) error
	Delete(id int64) error
	ListByPort(portID int64, page, pageSize int) ([]model.Berth, int64, error)
	List(page, pageSize int) ([]model.Berth, int64, error)
}

type berthDAOImpl struct {
	db *gorm.DB
}

func NewBerthDAO(db *gorm.DB) BerthDAO {
	return &berthDAOImpl{db: db}
}

func (d *berthDAOImpl) Create(berth *model.Berth) error {
	return d.db.Create(berth).Error
}

func (d *berthDAOImpl) GetByID(id int64) (*model.Berth, error) {
	var berth model.Berth
	err := d.db.Scopes(NotDeleted).First(&berth, id).Error
	return &berth, err
}

func (d *berthDAOImpl) Update(berth *model.Berth) error {
	return d.db.Save(berth).Error
}

func (d *berthDAOImpl) Delete(id int64) error {
	return d.db.Model(&model.Berth{}).
		Where("berth_id = ?", id).
		Update("delete_time", gorm.Expr("NOW()")).Error
}

func (d *berthDAOImpl) ListByPort(portID int64, page, pageSize int) ([]model.Berth, int64, error) {
	var berths []model.Berth
	var total int64
	query := d.db.Model(&model.Berth{}).Scopes(NotDeleted).Where("port_id = ?", portID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Find(&berths).Error
	return berths, total, err
}

func (d *berthDAOImpl) List(page, pageSize int) ([]model.Berth, int64, error) {
	var berths []model.Berth
	var total int64
	query := d.db.Model(&model.Berth{}).Scopes(NotDeleted)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Find(&berths).Error
	return berths, total, err
}
