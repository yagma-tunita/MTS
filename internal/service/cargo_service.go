package service

import (
	"context"

	"backend/internal/dao"
	"backend/internal/model"
)

type CargoService interface {
	ListAllCargos(ctx context.Context, page, pageSize int, keyword string) ([]model.OrderCargo, int64, error)
	CreateCargo(ctx context.Context, cargo *model.OrderCargo) error
	DeleteCargo(ctx context.Context, id int64) error
}

type cargoServiceImpl struct {
	dao dao.OrderCargoDAO
}

func NewCargoService(dao dao.OrderCargoDAO) CargoService {
	return &cargoServiceImpl{dao: dao}
}

func (s *cargoServiceImpl) ListAllCargos(ctx context.Context, page, pageSize int, keyword string) ([]model.OrderCargo, int64, error) {
	return s.dao.ListAll(page, pageSize, keyword)
}

func (s *cargoServiceImpl) CreateCargo(ctx context.Context, cargo *model.OrderCargo) error {
	return s.dao.Create(cargo)
}

func (s *cargoServiceImpl) DeleteCargo(ctx context.Context, id int64) error {
	return s.dao.Delete(id)
}
