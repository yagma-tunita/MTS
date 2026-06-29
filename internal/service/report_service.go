package service

import (
	"context"
	"time"

	"backend/internal/model"

	"gorm.io/gorm"
)

type ReportService interface {
	OrderStatistics(ctx context.Context, startDate, endDate time.Time) (*OrderStats, error)
	VoyageUtilization(ctx context.Context, lineID, vesselID int64, voyageDate time.Time) (*VoyageUtilization, error)
}

type OrderStats struct {
	TotalOrders int64   `json:"total_orders"`
	TotalWeight float64 `json:"total_weight"`
	TotalVolume float64 `json:"total_volume"`
	TotalCost   float64 `json:"total_cost"`
	Completed   int64   `json:"completed"`
	Cancelled   int64   `json:"cancelled"`
	InTransit   int64   `json:"in_transit"`
}

type VoyageUtilization struct {
	MaxCapacity float64 `json:"max_capacity"`
	Occupied    float64 `json:"occupied"`
	Utilization float64 `json:"utilization"` // percent
}

type reportServiceImpl struct {
	db *gorm.DB
}

func NewReportService(db *gorm.DB) ReportService {
	return &reportServiceImpl{db: db}
}

func (s *reportServiceImpl) OrderStatistics(ctx context.Context, startDate, endDate time.Time) (*OrderStats, error) {
	var stats OrderStats
	query := s.db.Model(&model.ShippingOrder{}).
		Where("delete_time IS NULL").
		Where("create_time BETWEEN ? AND ?", startDate, endDate)

	if err := query.Count(&stats.TotalOrders).Error; err != nil {
		return nil, err
	}
	if err := query.Select("COALESCE(SUM(total_weight_ton), 0)").Scan(&stats.TotalWeight).Error; err != nil {
		return nil, err
	}
	if err := query.Select("COALESCE(SUM(total_volume_cubic_meter), 0)").Scan(&stats.TotalVolume).Error; err != nil {
		return nil, err
	}
	if err := query.Select("COALESCE(SUM(total_cost), 0)").Scan(&stats.TotalCost).Error; err != nil {
		return nil, err
	}
	var statusCounts []struct {
		Status int8
		Count  int64
	}
	if err := query.Select("order_status, COUNT(*)").Group("order_status").Scan(&statusCounts).Error; err != nil {
		return nil, err
	}
	for _, sc := range statusCounts {
		switch sc.Status {
		case 3:
			stats.Completed = sc.Count
		case 4:
			stats.Cancelled = sc.Count
		case 2:
			stats.InTransit = sc.Count
		}
	}
	return &stats, nil
}

func (s *reportServiceImpl) VoyageUtilization(ctx context.Context, lineID, vesselID int64, voyageDate time.Time) (*VoyageUtilization, error) {
	var vessel model.Vessel
	if err := s.db.First(&vessel, vesselID).Error; err != nil {
		return nil, err
	}
	maxCap := float64(0)
	if vessel.MaxDeadweightTon != nil {
		maxCap = *vessel.MaxDeadweightTon
	}
	var occupied float64
	if err := s.db.Model(&model.SegmentCapacityUsage{}).
		Where("line_id = ? AND vessel_id = ? AND voyage_date = ?", lineID, vesselID, voyageDate).
		Select("COALESCE(SUM(occupied_ton), 0)").
		Scan(&occupied).Error; err != nil {
		return nil, err
	}
	utilization := 0.0
	if maxCap > 0 {
		utilization = occupied / maxCap * 100
	}
	return &VoyageUtilization{
		MaxCapacity: maxCap,
		Occupied:    occupied,
		Utilization: utilization,
	}, nil
}
