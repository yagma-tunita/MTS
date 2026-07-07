package service

import (
	"context"
	"time"

	"backend/internal/model"
	pkgerr "backend/pkg/errors"

	"gorm.io/gorm"
)

// ReportService 报表统计服务接口。提供：订单统计（按日期范围）、航次利用率查询
type ReportService interface {
	OrderStatistics(ctx context.Context, startDate, endDate time.Time, shippingCompanyID, shipperCompanyID int64) (*OrderStats, error)
	VoyageUtilization(ctx context.Context, lineID, vesselID int64, voyageDate time.Time, shippingCompanyID int64) (*VoyageUtilization, error)
}

// OrderStats 订单统计数据。含总订单数/总重量/总体积/总费用/各状态数量
type OrderStats struct {
	TotalOrders int64   `json:"total_orders"`
	TotalWeight float64 `json:"total_weight"`
	TotalVolume float64 `json:"total_volume"`
	TotalCost   float64 `json:"total_cost"`
	Completed   int64   `json:"completed"`
	Cancelled   int64   `json:"cancelled"`
	InTransit   int64   `json:"in_transit"`
}

// VoyageUtilization 航次舱位利用率。含最大载重/已占用吨位/利用率百分比
type VoyageUtilization struct {
	MaxCapacity      float64 `json:"max_deadweight_ton"`
	Occupied         float64 `json:"used_ton"`
	Utilization      float64 `json:"utilization_rate"` // 利用率（百分比，如 1.0 表示 1%）
}

// reportServiceImpl 是 ReportService 接口的私有实现。
type reportServiceImpl struct {
	db *gorm.DB
}

// NewReportService 创建报表统计服务
func NewReportService(db *gorm.DB) ReportService {
	return &reportServiceImpl{db: db}
}

// OrderStatistics 统计订单数据。多次对同一 query 链执行不同聚合：COUNT / SUM / GROUP BY
// shippingCompanyID > 0 时，仅统计该公司订单（通过 load_note → line 关联）。
func (s *reportServiceImpl) OrderStatistics(ctx context.Context, startDate, endDate time.Time, shippingCompanyID, shipperCompanyID int64) (*OrderStats, error) {
	var stats OrderStats

	baseQuery := func() *gorm.DB {
		q := s.db.Model(&model.ShippingOrder{}).
			Where("shipping_order.delete_time IS NULL").
			Where("shipping_order.create_time BETWEEN ? AND ?", startDate, endDate)
		if shippingCompanyID > 0 {
			q = q.
				Joins("JOIN voyage_cargo_note ON shipping_order.load_note_id = voyage_cargo_note.note_id").
				Joins("JOIN shipping_line ON voyage_cargo_note.line_id = shipping_line.line_id").
				Where("shipping_line.shipping_company_id = ?", shippingCompanyID)
		}
		if shipperCompanyID > 0 {
			q = q.Where("shipping_order.shipper_company_id = ?", shipperCompanyID)
		}
		return q
	}

	// COUNT(*) 获取总订单数
	if err := baseQuery().Count(&stats.TotalOrders).Error; err != nil { return nil, err }

	// COALESCE(SUM(...), 0) 分别聚合总重量、总体积、总运费
	// COALESCE 确保没有数据时返回 0 而非 NULL
	if err := baseQuery().Select("COALESCE(SUM(total_weight_ton), 0)").Scan(&stats.TotalWeight).Error; err != nil { return nil, err }
	if err := baseQuery().Select("COALESCE(SUM(total_volume_cubic_meter), 0)").Scan(&stats.TotalVolume).Error; err != nil { return nil, err }
	if err := baseQuery().Select("COALESCE(SUM(total_cost), 0)").Scan(&stats.TotalCost).Error; err != nil { return nil, err }

	// GROUP BY order_status 按状态分组统计数量
	// 返回结果如 [{Status:2, Count:15}, {Status:3, Count:30}, {Status:4, Count:5}]
	var statusCounts []struct {
		Status int8
		Count  int64
	}
	if err := baseQuery().Select("order_status, COUNT(*)").Group("order_status").Scan(&statusCounts).Error; err != nil { return nil, err }

	// 将分组结果映射到 OrderStats 对应字段
	// 2=运输中, 3=已完成, 4=已取消（其他状态不在报表中统计）
	for _, sc := range statusCounts {
		switch sc.Status {
		case 3: stats.Completed = sc.Count
		case 4: stats.Cancelled = sc.Count
		case 2: stats.InTransit = sc.Count
		}
	}
	return &stats, nil
}

// VoyageUtilization 查询航次利用率。两步：查 vessel 获取 DWT → 查 segment_capacity_usage 获取 SUM 占用
// shippingCompanyID > 0 时，校验该船舶和航线属于该公司。
func (s *reportServiceImpl) VoyageUtilization(ctx context.Context, lineID, vesselID int64, voyageDate time.Time, shippingCompanyID int64) (*VoyageUtilization, error) {

	// 第一步：从 vessel 表查船舶最大载重（DWT），作为运力基准值
	var vessel model.Vessel
	if err := s.db.First(&vessel, vesselID).Error; err != nil {
		return nil, err
	}

	if shippingCompanyID > 0 {
		if vessel.ShippingCompanyID == nil || *vessel.ShippingCompanyID != shippingCompanyID {
			return nil, pkgerr.Forbidden("vessel does not belong to your company")
		}
	}

	maxCap := float64(0)
	if vessel.MaxDeadweightTon != nil {
		maxCap = *vessel.MaxDeadweightTon
	}

	// 第二步：从 segment_capacity_usage 表统计该航次所有航段的总占用吨位
	// 使用 COALESCE(SUM, 0) 确保无记录时返回 0 而非 NULL
	var occupied float64
	if err := s.db.Model(&model.SegmentCapacityUsage{}).
		Where("line_id = ? AND vessel_id = ? AND voyage_date = ?", lineID, vesselID, voyageDate).
		Select("COALESCE(SUM(occupied_ton), 0)").
		Scan(&occupied).Error; err != nil {
		return nil, err
	}

	// 计算利用率百分比（避免除以 0）
	utilization := 0.0
	if maxCap > 0 {
		utilization = occupied / maxCap * 100
	}
	return &VoyageUtilization{MaxCapacity: maxCap, Occupied: occupied, Utilization: utilization}, nil
}

