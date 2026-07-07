package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"backend/internal/biz"
	"backend/internal/dao"
	"backend/internal/model"
	"backend/pkg/cache"
	pkgerr "backend/pkg/errors"

	"gorm.io/gorm"
)

// VoyageService 航次服务接口。
//
// 提供的功能：
//   - GetRemainingCapacity: 查询指定航段的剩余运力（用于前端展示或校验）。
//   - RecommendVoyages: 根据起止港口和需求吨位，推荐可用航次并按容量排序。
type VoyageService interface {
	GetRemainingCapacity(ctx context.Context, lineID, vesselID int64, voyageDate string, startPortID, endPortID int64) (float64, error)
	RecommendVoyages(ctx context.Context, startPortID, endPortID int64, requiredTon float64) ([]VoyageRecommendation, error)
	CreateVoyageBerthing(ctx context.Context, berthing *model.VoyageBerthing, unitPrice *float64) error
}

// VoyageRecommendation 航次推荐结果的一条记录。
type VoyageRecommendation struct {
	LineID            int64   // 航线 ID（查询详情用）
	VesselID          int64   // 船舶 ID（查询详情用）
	VoyageDate        string  // 航次日期
	VesselName        string  // 船舶名称（展示用）
	LineName          string  // 航线名称（展示用）
	RemainingCapacity float64 // 该航次所有航段中的最小剩余容量（瓶颈值）
}

// voyageServiceImpl 航次服务实现。
//
// 依赖的组件（注入方式：构造函数参数）：
//   - db: 直连数据库，用于查询所有航线和航次的复杂查询（多表关联、子查询）。
//     为什么不用 DAO？DAO 提供的是单表 CRUD，而 RecommendVoyages 需要跨表
//     JOIN 和子查询，直接使用 *gorm.DB 更灵活。
//   - shippingLineDAO: 查询航线信息（港口序列 JSON、航线名称等）。
//   - vesselDAO: 查询船舶信息（名称、最大载重吨 DWT）。
//   - voyageCargoNoteDAO: 查询航次货物记录（确认一个航线+船舶+日期是否存在航次）。
//   - segmentUsageDAO: 查询航段已占用的吨位，用于计算剩余容量。
//   - portSeqParser: biz 组件。将 JSON 格式的港口序列字符串解析为 []int64。
//   - voyageRecommender: biz 组件。推荐引擎，执行推荐算法（遍历·筛选·排序）。
type voyageServiceImpl struct {
	db                 *gorm.DB
	shippingLineDAO    dao.ShippingLineDAO
	vesselDAO          dao.VesselDAO
	voyageCargoNoteDAO dao.VoyageCargoNoteDAO
	segmentUsageDAO    dao.SegmentCapacityUsageDAO
	portSeqParser      biz.PortSequenceParser
	voyageRecommender  biz.VoyageRecommender
}

// NewVoyageService 创建航次服务实例
func NewVoyageService(
	db *gorm.DB,
	shippingLineDAO dao.ShippingLineDAO,
	vesselDAO dao.VesselDAO,
	voyageCargoNoteDAO dao.VoyageCargoNoteDAO,
	segmentUsageDAO dao.SegmentCapacityUsageDAO,
	portSeqParser biz.PortSequenceParser,
	voyageRecommender biz.VoyageRecommender,
) VoyageService {
	return &voyageServiceImpl{
		db:                 db,
		shippingLineDAO:    shippingLineDAO,
		vesselDAO:          vesselDAO,
		voyageCargoNoteDAO: voyageCargoNoteDAO,
		segmentUsageDAO:    segmentUsageDAO,
		portSeqParser:      portSeqParser,
		voyageRecommender:  voyageRecommender,
	}
}

// GetRemainingCapacity 查询指定航段的剩余运力。
//
// 计算方式：船舶最大载重（DWT）- 该航段已被占用的吨位。
// 其中已被占用的吨位来自 segment_capacity_usage 表的 SUM(occupied_ton)。
func (s *voyageServiceImpl) GetRemainingCapacity(ctx context.Context, lineID, vesselID int64, voyageDate string, startPortID, endPortID int64) (float64, error) {
	logger := Logger.With("method", "GetRemainingCapacity", "line_id", lineID, "vessel_id", vesselID, "date", voyageDate)
	logger.Debug("checking remaining capacity")

	vessel, err := s.vesselDAO.GetByID(vesselID)
	if err != nil || vessel.MaxDeadweightTon == nil {
		logger.Error("vessel not found", "vessel_id", vesselID)
		return 0, pkgerr.NotFound("vessel not found or max deadweight missing")
	}
	max := *vessel.MaxDeadweightTon
	used, err := s.segmentUsageDAO.GetOccupiedTons(lineID, vesselID, voyageDate, startPortID, endPortID)
	if err != nil {
		logger.Error("failed to get occupied tons", "error", err)
		return 0, err
	}
	remaining := max - used
	logger.Debug("remaining capacity", "value", remaining)
	return remaining, nil
}

// RecommendVoyages 推荐可用航次。流程：查缓存→遍历航线+航次→voyageRecommender 排序→写缓存
//
// 为什么要先查 voyage_cargo_note 再查 vessel：
//   先查 cargo note 可以快速过滤掉没有航次计划的航线+船舶组合，
//   避免查了船舶信息后才发现该船没有排班（减少不必要的数据库查询）。
func (s *voyageServiceImpl) RecommendVoyages(ctx context.Context, startPortID, endPortID int64, requiredTon float64) ([]VoyageRecommendation, error) {
	logger := Logger.With("method", "RecommendVoyages", "start_port", startPortID, "end_port", endPortID, "required_ton", requiredTon)
	logger.Info("recommending voyages")

	// ═══ 第一步：查缓存 ═══
	// key 格式 "voyage_rec:1:3:500.00"，包含起止港 ID 和需求吨数
	// 缓存 TTL 1 分钟。创建/取消订单时会 DeletePrefix("voyage_rec:") 清除
	cacheKey := fmt.Sprintf("voyage_rec:%d:%d:%.2f", startPortID, endPortID, requiredTon)
	if cached, found := cache.Get(cacheKey); found {
		if recs, ok := cached.([]VoyageRecommendation); ok {
			logger.Debug("cache hit", "key", cacheKey)
			return recs, nil
		}
		cache.Delete(cacheKey)
	}

	ctx, cancel := WithTimeout(ctx)
	defer cancel()

	// ═══ 第二步：加载所有航线 ═══
	// 只查未软删除的航线。每条航线包含 port_sequence(JSON) 用于解析港口顺序
	var lines []model.ShippingLine
	if err := s.db.WithContext(ctx).Where("delete_time IS NULL").Find(&lines).Error; err != nil {
		logger.Error("failed to load shipping lines", "error", err)
		return nil, err
	}

	var voyageInfos []biz.VoyageInfo
	type vesselInfo struct {
		Name      string
		MaxWeight float64
	}
	vesselCache := make(map[int64]vesselInfo) // 缓存船舶信息，避免重复查 DB

	// ═══ 第三步：遍历每条航线，收集有航次计划的 (航线+船舶+日期) 组合 ═══
	for _, line := range lines {
		// 跳过港口序列为空的航线（无法确定途径港口，无法推荐）
		if line.PortSequence == nil { continue }
		portIDs, err := s.portSeqParser.Parse(*line.PortSequence)
		if err != nil {
			logger.Warn("skip line due to parse error", "line_id", line.LineID, "error", err)
			continue
		}

		// 从 voyage_cargo_note 表查 DISTINCT 的 (line_id, vessel_id, voyage_date)
		// 只有有 cargo note 的航次才纳入推荐（即有实际的装/卸货计划）
		var voyages []struct {
			VesselID   int64
			VoyageDate string
		}
		if err := s.db.WithContext(ctx).Table("voyage_cargo_note").
			Select("DISTINCT vessel_id, voyage_date").
			Where("line_id = ?", line.LineID).
			Scan(&voyages).Error; err != nil {
			logger.Warn("skip line due to voyage query error", "line_id", line.LineID, "error", err)
			continue
		}

		// 遍历每个航次，拼装 VoyageInfo（含船舶名、最大载重、港口序列）
		for _, v := range voyages {
			vi, ok := vesselCache[v.VesselID]
			if !ok {
				vessel, err := s.vesselDAO.GetByID(v.VesselID)
				if err != nil {
					logger.Warn("skip voyage due to vessel not found", "vessel_id", v.VesselID, "error", err)
					continue
				}
				vi.Name = vessel.VesselName
				if vessel.MaxDeadweightTon != nil {
					vi.MaxWeight = *vessel.MaxDeadweightTon
				}
				vesselCache[v.VesselID] = vi
			}
			voyageInfos = append(voyageInfos, biz.VoyageInfo{
				LineID:     line.LineID,
				VesselID:   v.VesselID,
				VoyageDate: v.VoyageDate,
				VesselName: vi.Name,
				LineName:   line.LineName,
				MaxWeight:  vi.MaxWeight,
				PortIDs:    portIDs,
			})
		}
	}

	getRemaining := func(lineID, vesselID int64, voyageDate string, startPortID, endPortID int64) (float64, error) {
		return s.GetRemainingCapacity(ctx, lineID, vesselID, voyageDate, startPortID, endPortID)
	}

	recommended, err := s.voyageRecommender.Recommend(voyageInfos, startPortID, endPortID, requiredTon, getRemaining)
	if err != nil {
		logger.Error("recommendation failed", "error", err)
		return nil, err
	}

	result := make([]VoyageRecommendation, len(recommended))
	for i, r := range recommended {
		// MySQL DATE 列被 Scan 到 string 时格式不确定（可能是 "2026-05-01T00:00:00Z" 等），
		// 统一转为 YYYY-MM-DD，保证前端能正确回传给创建订单接口。
		voyageDate := r.VoyageDate
		if len(voyageDate) >= 10 {
			if t, err := time.Parse("2006-01-02", voyageDate[:10]); err == nil {
				voyageDate = t.Format("2006-01-02")
			} else {
				voyageDate = voyageDate[:10]
			}
		}
		result[i] = VoyageRecommendation{
			LineID:            r.LineID,
			VesselID:          r.VesselID,
			VoyageDate:        voyageDate,
			VesselName:        r.VesselName,
			LineName:          r.LineName,
			RemainingCapacity: r.MinRemainingCap,
		}
	}

	cache.Set(cacheKey, result, 1*time.Minute)
	logger.Debug("cache stored", "key", cacheKey)

	logger.Info("recommendation completed", "count", len(result))
	return result, nil
}

// CreateVoyageBerthing 创建航次靠泊记录（船公司申请航线时使用）。
// 同时自动创建对应的航次货物通知单，使该航次能被推荐并接受订单。
func (s *voyageServiceImpl) CreateVoyageBerthing(ctx context.Context, berthing *model.VoyageBerthing, unitPrice *float64) error {
	err := s.db.Create(berthing).Error
	if err != nil {
		return err
	}

	// 为每个靠泊记录自动创建 cargo note，以便该航次出现在推荐列表中。
	cn := "待定"
	ct := "bulk"
	z := 0.0
	op := "LOAD"
	up := z
	if unitPrice != nil {
		up = *unitPrice
	}
	note := &model.VoyageCargoNote{
		LineID:           berthing.LineID,
		VesselID:         berthing.VesselID,
		VoyageDate:       berthing.VoyageDate,
		SequenceNo:       berthing.SequenceNo,
		CargoName:        &cn,
		CargoType:        &ct,
		Quantity:         &z,
		WeightTon:        &z,
		VolumeCubicMeter: &z,
		UnitPrice:        &up,
		Subtotal:         &z,
		OperationType:    &op,
		CreateTime:       time.Now(),
		UpdateTime:       time.Now(),
	}
	if createErr := s.db.Create(note).Error; createErr != nil {
		log.Printf("warn: failed to auto-create cargo note: line=%d vessel=%d date=%s seq=%d err=%v",
			*berthing.LineID, *berthing.VesselID, berthing.VoyageDate, berthing.SequenceNo, createErr)
	}
	return nil
}

