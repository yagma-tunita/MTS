package service

import (
	"context"

	"backend/internal/dao"
	"backend/internal/model"
)

type CargoService interface { // 货物服务接口
	ListAllCargos(ctx context.Context, page, pageSize int) ([]model.OrderCargo, int64, error) // 管理员查询所有货物运输记录
}

type cargoServiceImpl struct { // 货物服务实现
	dao dao.OrderCargoDAO
}

func NewCargoService(dao dao.OrderCargoDAO) CargoService { // 创建货物服务
	return &cargoServiceImpl{dao: dao}
}

func (s *cargoServiceImpl) ListAllCargos(ctx context.Context, page, pageSize int) ([]model.OrderCargo, int64, error) {
	return s.dao.ListAll(page, pageSize)
}
