package service

import (
	"context"
	"time"

	"backend/internal/dao"
	"backend/internal/model"
	pkgerr "backend/pkg/errors"
)

type VoyageBerthingService interface { // 航次靠泊服务接口
	UpdateActualTimes(ctx context.Context, berthingID int64, actualArrival, actualDeparture *time.Time) error // 更新实际到达/出发时间
	ListByShippingCompany(ctx context.Context, companyID int64) ([]model.VoyageBerthing, error)
}

type voyageBerthingServiceImpl struct { // 航次靠泊服务实现
	dao dao.VoyageBerthingDAO
}

func NewVoyageBerthingService(dao dao.VoyageBerthingDAO) VoyageBerthingService { // 创建航次靠泊服务
	return &voyageBerthingServiceImpl{dao: dao}
}

func (s *voyageBerthingServiceImpl) UpdateActualTimes(ctx context.Context, berthingID int64, actualArrival, actualDeparture *time.Time) error {
	berthing, err := s.dao.GetByID(berthingID)
	if err != nil {
		return pkgerr.NotFound("berthing record not found")
	}
	berthing.ActualArrivalTime = actualArrival
	berthing.ActualDepartureTime = actualDeparture
	return s.dao.Update(berthing)
}

func (s *voyageBerthingServiceImpl) ListByShippingCompany(ctx context.Context, companyID int64) ([]model.VoyageBerthing, error) {
	return s.dao.ListByShippingCompany(companyID)
}
