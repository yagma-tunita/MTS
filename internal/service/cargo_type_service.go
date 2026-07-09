package service

import (
	"context"

	"backend/internal/dao"
	"backend/internal/model"
)

type CargoTypeService interface {
	ListAll(ctx context.Context) ([]model.CargoType, error)
	List(ctx context.Context, page, pageSize int, keyword string) ([]model.CargoType, int64, error)
	Create(ctx context.Context, t *model.CargoType) error
	Update(ctx context.Context, t *model.CargoType) error
	Delete(ctx context.Context, id int64) error
}

type cargoTypeServiceImpl struct {
	dao dao.CargoTypeDAO
}

func NewCargoTypeService(dao dao.CargoTypeDAO) CargoTypeService {
	return &cargoTypeServiceImpl{dao: dao}
}

func (s *cargoTypeServiceImpl) ListAll(ctx context.Context) ([]model.CargoType, error) {
	return s.dao.ListAll()
}

func (s *cargoTypeServiceImpl) List(ctx context.Context, page, pageSize int, keyword string) ([]model.CargoType, int64, error) {
	return s.dao.List(page, pageSize, keyword)
}

func (s *cargoTypeServiceImpl) Create(ctx context.Context, t *model.CargoType) error {
	return s.dao.Create(t)
}

func (s *cargoTypeServiceImpl) Update(ctx context.Context, t *model.CargoType) error {
	return s.dao.Update(t)
}

func (s *cargoTypeServiceImpl) Delete(ctx context.Context, id int64) error {
	return s.dao.Delete(id)
}
