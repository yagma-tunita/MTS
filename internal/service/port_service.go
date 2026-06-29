package service

import (
	"context"
	"fmt"
	"time"

	"backend/internal/dao"
	"backend/internal/model"
	"backend/pkg/cache"
	pkgerr "backend/pkg/errors"
)

type PortService interface {
	GetPortByID(ctx context.Context, id int64) (*model.Port, error)
	ListPorts(ctx context.Context, page, pageSize int) ([]model.Port, int64, error)
	ListPortsByCity(ctx context.Context, cityID int64, page, pageSize int) ([]model.Port, int64, error)
}

type portServiceImpl struct {
	dao dao.PortDAO
}

func NewPortService(dao dao.PortDAO) PortService {
	return &portServiceImpl{dao: dao}
}

func (s *portServiceImpl) GetPortByID(ctx context.Context, id int64) (*model.Port, error) {
	logger := Logger.With("method", "GetPortByID", "port_id", id)
	logger.Debug("fetching port")

	cacheKey := fmt.Sprintf("port:id:%d", id)
	if cached, found := cache.Get(cacheKey); found {
		if port, ok := cached.(*model.Port); ok {
			logger.Debug("cache hit", "key", cacheKey)
			return port, nil
		}
		// Invalid data, delete cache
		cache.Delete(cacheKey)
	}

	port, err := s.dao.GetByID(id)
	if err != nil {
		logger.Error("port not found", "error", err)
		return nil, pkgerr.NotFound("port not found")
	}

	// Cache for 10 minutes (ports are rarely changed)
	cache.Set(cacheKey, port, 10*time.Minute)
	return port, nil
}

func (s *portServiceImpl) ListPorts(ctx context.Context, page, pageSize int) ([]model.Port, int64, error) {
	logger := Logger.With("method", "ListPorts", "page", page, "page_size", pageSize)
	logger.Debug("listing ports")

	cacheKey := fmt.Sprintf("ports:list:%d:%d", page, pageSize)
	if cached, found := cache.Get(cacheKey); found {
		if result, ok := cached.(struct {
			Ports []model.Port
			Total int64
		}); ok {
			logger.Debug("cache hit", "key", cacheKey)
			return result.Ports, result.Total, nil
		}
		cache.Delete(cacheKey)
	}

	ports, total, err := s.dao.List(page, pageSize)
	if err != nil {
		logger.Error("failed to list ports", "error", err)
		return nil, 0, err
	}

	// Cache for 10 minutes
	cache.Set(cacheKey, struct {
		Ports []model.Port
		Total int64
	}{ports, total}, 10*time.Minute)

	logger.Debug("ports listed", "count", len(ports), "total", total)
	return ports, total, nil
}

func (s *portServiceImpl) ListPortsByCity(ctx context.Context, cityID int64, page, pageSize int) ([]model.Port, int64, error) {
	logger := Logger.With("method", "ListPortsByCity", "city_id", cityID, "page", page, "page_size", pageSize)
	logger.Debug("listing ports by city")

	cacheKey := fmt.Sprintf("ports:city:%d:%d:%d", cityID, page, pageSize)
	if cached, found := cache.Get(cacheKey); found {
		if result, ok := cached.(struct {
			Ports []model.Port
			Total int64
		}); ok {
			logger.Debug("cache hit", "key", cacheKey)
			return result.Ports, result.Total, nil
		}
		cache.Delete(cacheKey)
	}

	ports, total, err := s.dao.ListByCity(cityID, page, pageSize)
	if err != nil {
		logger.Error("failed to list ports by city", "error", err)
		return nil, 0, err
	}

	cache.Set(cacheKey, struct {
		Ports []model.Port
		Total int64
	}{ports, total}, 10*time.Minute)

	logger.Debug("ports listed", "count", len(ports), "total", total)
	return ports, total, nil
}
