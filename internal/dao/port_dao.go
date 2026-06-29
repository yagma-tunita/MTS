package dao

import (
	"backend/internal/model"

	"gorm.io/gorm"
)

type PortDAO interface {
	Create(port *model.Port) error
	GetByID(id int64) (*model.Port, error)
	GetByCode(code string) (*model.Port, error)
	Update(port *model.Port) error
	Delete(id int64) error
	ListByCity(cityID int64, page, pageSize int) ([]model.Port, int64, error)
	List(page, pageSize int) ([]model.Port, int64, error)
}

type portDAOImpl struct {
	db *gorm.DB
}

func NewPortDAO(db *gorm.DB) PortDAO {
	return &portDAOImpl{db: db}
}

func (d *portDAOImpl) Create(port *model.Port) error {
	return d.db.Create(port).Error
}

func (d *portDAOImpl) GetByID(id int64) (*model.Port, error) {
	var port model.Port
	err := d.db.Scopes(NotDeleted).First(&port, id).Error
	return &port, err
}

func (d *portDAOImpl) GetByCode(code string) (*model.Port, error) {
	var port model.Port
	err := d.db.Scopes(NotDeleted).Where("port_code = ?", code).First(&port).Error
	return &port, err
}

func (d *portDAOImpl) Update(port *model.Port) error {
	return d.db.Save(port).Error
}

func (d *portDAOImpl) Delete(id int64) error {
	return d.db.Model(&model.Port{}).
		Where("port_id = ?", id).
		Update("delete_time", gorm.Expr("NOW()")).Error
}

func (d *portDAOImpl) ListByCity(cityID int64, page, pageSize int) ([]model.Port, int64, error) {
	var ports []model.Port
	var total int64
	query := d.db.Model(&model.Port{}).Scopes(NotDeleted).Where("city_id = ?", cityID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Find(&ports).Error
	return ports, total, err
}

func (d *portDAOImpl) List(page, pageSize int) ([]model.Port, int64, error) {
	var ports []model.Port
	var total int64
	query := d.db.Model(&model.Port{}).Scopes(NotDeleted)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Find(&ports).Error
	return ports, total, err
}
