package dao

import (
	"backend/internal/model"

	"gorm.io/gorm"
)

type AdminDAO interface {
	Create(admin *model.Admin) error
	GetByID(id int64) (*model.Admin, error)
	GetByUsername(username string) (*model.Admin, error)
	Update(admin *model.Admin) error
	Delete(id int64) error
	List(page, pageSize int) ([]model.Admin, int64, error)
}

type adminDAOImpl struct {
	db *gorm.DB
}

func NewAdminDAO(db *gorm.DB) AdminDAO {
	return &adminDAOImpl{db: db}
}

func (d *adminDAOImpl) Create(admin *model.Admin) error {
	return d.db.Create(admin).Error
}

func (d *adminDAOImpl) GetByID(id int64) (*model.Admin, error) {
	var admin model.Admin
	err := d.db.Scopes(NotDeleted).First(&admin, id).Error
	return &admin, err
}

func (d *adminDAOImpl) GetByUsername(username string) (*model.Admin, error) {
	var admin model.Admin
	err := d.db.Scopes(NotDeleted).
		Where("username = ?", username).
		First(&admin).Error
	return &admin, err
}

func (d *adminDAOImpl) Update(admin *model.Admin) error {
	return d.db.Save(admin).Error
}

func (d *adminDAOImpl) Delete(id int64) error {
	return d.db.Model(&model.Admin{}).
		Where("admin_id = ?", id).
		Update("delete_time", gorm.Expr("NOW()")).Error
}

func (d *adminDAOImpl) List(page, pageSize int) ([]model.Admin, int64, error) {
	var admins []model.Admin
	var total int64
	query := d.db.Model(&model.Admin{}).Scopes(NotDeleted)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Find(&admins).Error
	return admins, total, err
}
