package service

import (
	"context"
	"fmt"
	"time"

	"backend/internal/biz"
	"backend/internal/dao"
	"backend/internal/model"
	"backend/pkg/cache"
	pkgerr "backend/pkg/errors"

	"gorm.io/gorm"
)

type VoyageService interface {
	GetRemainingCapacity(ctx context.Context, lineID, vesselID int64, voyageDate string, startPortID, endPortID int64) (float64, error)
	RecommendVoyages(ctx context.Context, startPortID, endPortID int64, requiredTon float64) ([]VoyageRecommendation, error)
}

type VoyageRecommendation struct {
	LineID            int64
	VesselID          int64
	VoyageDate        string
	VesselName        string
	LineName          string
	RemainingCapacity float64
}

type voyageServiceImpl struct {
	db                 *gorm.DB
	shippingLineDAO    dao.ShippingLineDAO
	vesselDAO          dao.VesselDAO
	voyageCargoNoteDAO dao.VoyageCargoNoteDAO
	segmentUsageDAO    dao.SegmentCapacityUsageDAO
	portSeqParser      biz.PortSequenceParser
	voyageRecommender  biz.VoyageRecommender
}

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

func (s *voyageServiceImpl) RecommendVoyages(ctx context.Context, startPortID, endPortID int64, requiredTon float64) ([]VoyageRecommendation, error) {
	logger := Logger.With("method", "RecommendVoyages", "start_port", startPortID, "end_port", endPortID, "required_ton", requiredTon)
	logger.Info("recommending voyages")

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

	var lines []model.ShippingLine
	if err := s.db.WithContext(ctx).Where("delete_time IS NULL").Find(&lines).Error; err != nil {
		logger.Error("failed to load shipping lines", "error", err)
		return nil, err
	}

	var voyageInfos []biz.VoyageInfo
	vesselNameMap := make(map[int64]string)

	for _, line := range lines {
		if line.PortSequence == nil {
			continue
		}
		portIDs, err := s.portSeqParser.Parse(*line.PortSequence)
		if err != nil {
			logger.Warn("skip line due to parse error", "line_id", line.LineID, "error", err)
			continue
		}
		var voyages []struct {
			VesselID   int64
			VoyageDate string
		}
		s.db.WithContext(ctx).Table("voyage_cargo_note").
			Select("DISTINCT vessel_id, voyage_date").
			Where("line_id = ?", line.LineID).
			Scan(&voyages)

		for _, v := range voyages {
			if _, ok := vesselNameMap[v.VesselID]; !ok {
				vessel, err := s.vesselDAO.GetByID(v.VesselID)
				if err == nil {
					vesselNameMap[v.VesselID] = vessel.VesselName
				}
			}
			vesselName := vesselNameMap[v.VesselID]
			vesselObj, _ := s.vesselDAO.GetByID(v.VesselID)
			maxWeight := float64(0)
			if vesselObj != nil && vesselObj.MaxDeadweightTon != nil {
				maxWeight = *vesselObj.MaxDeadweightTon
			}
			voyageInfos = append(voyageInfos, biz.VoyageInfo{
				LineID:     line.LineID,
				VesselID:   v.VesselID,
				VoyageDate: v.VoyageDate,
				VesselName: vesselName,
				LineName:   line.LineName,
				MaxWeight:  maxWeight,
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
		result[i] = VoyageRecommendation{
			LineID:            r.LineID,
			VesselID:          r.VesselID,
			VoyageDate:        r.VoyageDate,
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
