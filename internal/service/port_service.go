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

// PortService 港口服务接口，支持查询单个、列表、按城市查询、增删改，带缓存。
type PortService interface {
	GetPortByID(ctx context.Context, id int64) (*model.Port, error)
	ListPorts(ctx context.Context, page, pageSize int, keyword string) ([]model.Port, int64, error)
	ListPortsByCity(ctx context.Context, cityID int64, page, pageSize int) ([]model.Port, int64, error)
	CreatePort(ctx context.Context, port *model.Port) error
	UpdatePort(ctx context.Context, port *model.Port) error
	DeletePort(ctx context.Context, id int64) error
}

// portServiceImpl 是 PortService 接口的私有实现。
type portServiceImpl struct {
	dao dao.PortDAO
}

// NewPortService 创建港口查询服务
func NewPortService(dao dao.PortDAO) PortService {
	return &portServiceImpl{dao: dao}
}

// GetPortByID 查询单个港口，使用缓存加速。缓存策略：先查 cache→命中直接返回→未命中查 DB→写入 cache
func (s *portServiceImpl) GetPortByID(ctx context.Context, id int64) (*model.Port, error) {
	logger := Logger.With("method", "GetPortByID", "port_id", id)
	logger.Debug("fetching port")

	// 先查缓存。key 格式 "port:id:1"，缓存的是 *model.Port
	cacheKey := fmt.Sprintf("port:id:%d", id)
	if cached, found := cache.Get(cacheKey); found {
		// 类型断言，防止缓存中存入了非 *model.Port 类型的数据
		if port, ok := cached.(*model.Port); ok {
			logger.Debug("cache hit", "key", cacheKey)
			return port, nil
		}
		// 类型断言失败，说明缓存数据异常，删除后从 DB 重新读取
		cache.Delete(cacheKey)
	}

	// 缓存未命中，从数据库查询
	port, err := s.dao.GetByID(id)
	if err != nil {
		logger.Error("port not found", "error", err)
		return nil, pkgerr.NotFound("port not found")
	}

	// 写入缓存，TTL 10 分钟。港口数据极少变更，10 分钟缓存可大幅减少 DB 查询
	cache.Set(cacheKey, port, 10*time.Minute)
	return port, nil
}

// ListPorts 分页查询港口列表（缓存版）。key 中带 page+pageSize，不同分页参数独立缓存
func (s *portServiceImpl) ListPorts(ctx context.Context, page, pageSize int, keyword string) ([]model.Port, int64, error) {
	logger := Logger.With("method", "ListPorts", "page", page, "page_size", pageSize)
	logger.Debug("listing ports")

	// key 格式 "ports:list:1:20"，page=1 pageSize=20 的缓存与其他分页不冲突
	cacheKey := fmt.Sprintf("ports:list:%d:%d", page, pageSize)
	if cached, found := cache.Get(cacheKey); found {
		// 匿名结构体类型断言，需要与写入时的类型完全一致
		if result, ok := cached.(struct {
			Ports []model.Port
			Total int64
		}); ok {
			logger.Debug("cache hit", "key", cacheKey)
			return result.Ports, result.Total, nil
		}
		cache.Delete(cacheKey)
	}

	// 缓存未命中，调 DAO 查询数据库
	ports, total, err := s.dao.List(page, pageSize, keyword)
	if err != nil {
		logger.Error("failed to list ports", "error", err)
		return nil, 0, err
	}

	// 将查询结果同时缓存起来，下次同样分页参数直接走缓存
	cache.Set(cacheKey, struct {
		Ports []model.Port
		Total int64
	}{ports, total}, 10*time.Minute)

	logger.Debug("ports listed", "count", len(ports), "total", total)
	return ports, total, nil
}

// CreatePort 创建新港口。
func (s *portServiceImpl) CreatePort(ctx context.Context, port *model.Port) error {
	return s.dao.Create(port)
}

// UpdatePort 更新港口信息，同时清除缓存确保下次读取到最新数据。
func (s *portServiceImpl) UpdatePort(ctx context.Context, port *model.Port) error {
	err := s.dao.Update(port)
	if err == nil {
		cache.Delete(fmt.Sprintf("port:id:%d", port.PortID))
		cache.Delete(fmt.Sprintf("ports:list:%d:%d", 1, 10))
		cache.Delete(fmt.Sprintf("ports:list:%d:%d", 2, 10))
	}
	return err
}

// DeletePort 软删除港口。
func (s *portServiceImpl) DeletePort(ctx context.Context, id int64) error {
	err := s.dao.Delete(id)
	if err == nil {
		cache.Delete(fmt.Sprintf("port:id:%d", id))
	}
	return err
}

// ListPortsByCity 按城市筛选港口（缓存版）。key 含 cityID，不同城市独立缓存
func (s *portServiceImpl) ListPortsByCity(ctx context.Context, cityID int64, page, pageSize int) ([]model.Port, int64, error) {
	logger := Logger.With("method", "ListPortsByCity", "city_id", cityID, "page", page, "page_size", pageSize)
	logger.Debug("listing ports by city")

	// key 格式 "ports:city:1:1:20" = 城市1 + 第1页 + 每页20条
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

