package service

import (
	"context"

	"backend/internal/dao"
	"backend/internal/model"
)

type CityService interface {
	ListCities(ctx context.Context, page, pageSize int, cityName string) ([]model.City, int64, error)
	CreateCity(ctx context.Context, city *model.City) error
	UpdateCity(ctx context.Context, city *model.City) error
	DeleteCity(ctx context.Context, id int64) error
}

type cityServiceImpl struct {
	dao dao.CityDAO
}

func NewCityService(dao dao.CityDAO) CityService {
	return &cityServiceImpl{dao: dao}
}

func (s *cityServiceImpl) ListCities(ctx context.Context, page, pageSize int, cityName string) ([]model.City, int64, error) {
	return s.dao.List(page, pageSize, cityName)
}

func (s *cityServiceImpl) CreateCity(ctx context.Context, city *model.City) error {
	return s.dao.Create(city)
}

func (s *cityServiceImpl) UpdateCity(ctx context.Context, city *model.City) error {
	return s.dao.Update(city)
}

func (s *cityServiceImpl) DeleteCity(ctx context.Context, id int64) error {
	return s.dao.Delete(id)
}
