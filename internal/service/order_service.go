package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"backend/internal/biz"
	"backend/internal/dao"
	"backend/internal/model"
	"backend/pkg/cache"
	"backend/pkg/config"
	pkgerr "backend/pkg/errors"

	"gorm.io/gorm"
)

type OrderService interface {
	CreateOrder(ctx context.Context, req *CreateOrderRequest) (*model.ShippingOrder, error)
	CancelOrder(ctx context.Context, orderID int64) error
	UpdateOrderStatus(ctx context.Context, orderID int64, newStatus int8) error
	GetOrderByID(ctx context.Context, orderID int64) (*model.ShippingOrder, error)
	ListOrdersByShipper(ctx context.Context, shipperCompanyID int64, req PageRequest) ([]model.ShippingOrder, int64, error)
	GetOrderTracking(ctx context.Context, orderID int64) (*OrderTracking, error)
}

type CreateOrderRequest struct {
	ShipperCompanyID  int64
	CityID            int64
	LineID            int64
	VesselID          int64
	VoyageDate        string
	StartPortID       int64
	EndPortID         int64
	CargoItems        []CargoItem
	ShipperContact    string
	ConsigneeContact  string
	ExpectedDeparture *string
	ExpectedArrival   *string
}

type CargoItem struct {
	CargoName  string  `json:"cargo_name"`
	CargoType  string  `json:"cargo_type"`
	Quantity   float64 `json:"quantity"`
	WeightTon  float64 `json:"weight_ton"`
	VolumeCubM float64 `json:"volume_cub_m"`
	UnitPrice  float64 `json:"unit_price"`
	Subtotal   float64 `json:"subtotal"`
}

type orderServiceImpl struct {
	db                 *gorm.DB
	orderDAO           dao.ShippingOrderDAO
	orderCargoDAO      dao.OrderCargoDAO
	segmentUsageDAO    dao.SegmentCapacityUsageDAO
	voyageCargoNoteDAO dao.VoyageCargoNoteDAO
	vesselDAO          dao.VesselDAO
	shippingLineDAO    dao.ShippingLineDAO
	portSeqParser      biz.PortSequenceParser
	segCalc            biz.SegmentCalculator
	capChecker         biz.CapacityChecker
	costCalc           biz.CostCalculator
	orderNoGen         biz.OrderNoGenerator
	stateMachine       biz.OrderStateMachine
	wsSvc              WebSocketService
}

func NewOrderService(
	db *gorm.DB,
	orderDAO dao.ShippingOrderDAO,
	orderCargoDAO dao.OrderCargoDAO,
	segmentUsageDAO dao.SegmentCapacityUsageDAO,
	voyageCargoNoteDAO dao.VoyageCargoNoteDAO,
	vesselDAO dao.VesselDAO,
	shippingLineDAO dao.ShippingLineDAO,
	portSeqParser biz.PortSequenceParser,
	segCalc biz.SegmentCalculator,
	capChecker biz.CapacityChecker,
	costCalc biz.CostCalculator,
	orderNoGen biz.OrderNoGenerator,
	stateMachine biz.OrderStateMachine,
	wsSvc WebSocketService,
) OrderService {
	return &orderServiceImpl{
		db:                 db,
		orderDAO:           orderDAO,
		orderCargoDAO:      orderCargoDAO,
		segmentUsageDAO:    segmentUsageDAO,
		voyageCargoNoteDAO: voyageCargoNoteDAO,
		vesselDAO:          vesselDAO,
		shippingLineDAO:    shippingLineDAO,
		portSeqParser:      portSeqParser,
		segCalc:            segCalc,
		capChecker:         capChecker,
		costCalc:           costCalc,
		orderNoGen:         orderNoGen,
		stateMachine:       stateMachine,
		wsSvc:              wsSvc,
	}
}

func (s *orderServiceImpl) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*model.ShippingOrder, error) {
	logger := Logger.With(
		"method", "CreateOrder",
		"shipper_id", req.ShipperCompanyID,
		"line_id", req.LineID,
		"vessel_id", req.VesselID,
	)
	logger.Info("creating order")

	ctx, cancel := WithTimeout(ctx)
	defer cancel()

	if len(req.CargoItems) == 0 {
		logger.Warn("empty cargo list")
		return nil, pkgerr.BadRequest("at least one cargo item required")
	}

	bizItems := make([]biz.CargoItem, len(req.CargoItems))
	for i, it := range req.CargoItems {
		bizItems[i] = biz.CargoItem{
			WeightTon: it.WeightTon,
			VolumeM3:  it.VolumeCubM,
			UnitPrice: it.UnitPrice,
			Quantity:  it.Quantity,
		}
	}

	costResult, err := s.costCalc.Calculate(bizItems)
	if err != nil {
		if errors.Is(err, biz.ErrEmptyCargoList) {
			return nil, pkgerr.BadRequest(err.Error())
		}
		logger.Error("cost calculation failed", "error", err)
		return nil, err
	}
	totalWeight := costResult.TotalWeightTon
	totalVolume := costResult.TotalVolumeM3

	vessel, err := s.vesselDAO.GetByID(req.VesselID)
	if err != nil || vessel.MaxDeadweightTon == nil {
		logger.Error("vessel not found", "vessel_id", req.VesselID)
		return nil, pkgerr.NotFound("vessel not found or max deadweight missing")
	}
	maxWeight := *vessel.MaxDeadweightTon

	line, err := s.shippingLineDAO.GetByID(req.LineID)
	if err != nil || line.PortSequence == nil {
		logger.Error("shipping line not found", "line_id", req.LineID)
		return nil, pkgerr.NotFound("shipping line not found or port sequence missing")
	}

	totalDistance := 0.0
	if line.TotalDistanceNm != nil {
		totalDistance = *line.TotalDistanceNm
	} else {
		logger.Warn("line total distance is nil, using 0", "line_id", req.LineID)
	}

	cfg := config.Get()
	baseRate := cfg.Freight.BaseRatePerTonNm
	cargoFactor := 1.0
	if len(req.CargoItems) > 0 && cfg.Freight.CargoTypeFactors != nil {
		cargoType := req.CargoItems[0].CargoType
		if factor, ok := cfg.Freight.CargoTypeFactors[cargoType]; ok {
			cargoFactor = factor
		}
	}
	totalCost := totalWeight * totalDistance * baseRate * cargoFactor

	portIDs, err := s.portSeqParser.Parse(*line.PortSequence)
	if err != nil {
		logger.Error("parse port sequence failed", "error", err)
		return nil, pkgerr.BadRequest("invalid port sequence")
	}

	segments, err := s.segCalc.Calculate(portIDs, req.StartPortID, req.EndPortID)
	if err != nil {
		if errors.Is(err, biz.ErrPortNotFoundInSeq) || errors.Is(err, biz.ErrStartAfterEnd) {
			return nil, pkgerr.BadRequest(err.Error())
		}
		return nil, err
	}

	loadNote, err := s.voyageCargoNoteDAO.FindByPortAndOp(req.LineID, req.VesselID, req.VoyageDate, req.StartPortID, "LOAD")
	if err != nil {
		logger.Error("load note not found", "port", req.StartPortID)
		return nil, pkgerr.NotFound("no LOAD cargo note for start port")
	}
	unloadNote, err := s.voyageCargoNoteDAO.FindByPortAndOp(req.LineID, req.VesselID, req.VoyageDate, req.EndPortID, "UNLOAD")
	if err != nil {
		logger.Error("unload note not found", "port", req.EndPortID)
		return nil, pkgerr.NotFound("no UNLOAD cargo note for end port")
	}

	var order *model.ShippingOrder
	voyageDateObj := MustParseDate(req.VoyageDate)
	lockName := VoyageLockKey(req.LineID, req.VesselID, req.VoyageDate)

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ok, err := AcquireLock(tx, lockName, 10)
		if err != nil || !ok {
			return pkgerr.New(pkgerr.CodeTooManyRequests, "failed to acquire lock, please retry")
		}
		defer ReleaseLock(tx, lockName)

		for _, seg := range segments {
			var dummy int
			if err := tx.Raw(`
                SELECT 1 FROM segment_capacity_usage
                WHERE line_id = ? AND vessel_id = ? AND voyage_date = ? AND start_port_id = ? AND end_port_id = ?
                FOR UPDATE
            `, req.LineID, req.VesselID, req.VoyageDate, seg[0], seg[1]).Scan(&dummy).Error; err != nil && err != gorm.ErrRecordNotFound {
				return err
			}
		}

		occupiedGetter := func(seg [2]int64) (float64, error) {
			return s.segmentUsageDAO.GetOccupiedTons(req.LineID, req.VesselID, req.VoyageDate, seg[0], seg[1])
		}
		ok, minRemaining, err := s.capChecker.Check(segments, maxWeight, occupiedGetter, totalWeight)
		if err != nil {
			return err
		}
		if !ok {
			return pkgerr.New(pkgerr.CodeConflict, fmt.Sprintf("insufficient capacity, remaining: %.2f", minRemaining))
		}

		orderNo := s.orderNoGen.Generate()

		order = &model.ShippingOrder{
			OrderNo:               orderNo,
			ShipperCompanyID:      &req.ShipperCompanyID,
			CityID:                &req.CityID,
			LoadNoteID:            &loadNote.NoteID,
			UnloadNoteID:          &unloadNote.NoteID,
			DeparturePortID:       &req.StartPortID,
			DestinationPortID:     &req.EndPortID,
			TotalWeightTon:        &totalWeight,
			TotalVolumeCubicMeter: &totalVolume,
			TotalCost:             &totalCost,
			ShipperContact:        &req.ShipperContact,
			ConsigneeContact:      &req.ConsigneeContact,
			PaymentStatus:         PtrInt8(0),
			OrderStatus:           PtrInt8(1),
		}
		if err := tx.Create(order).Error; err != nil {
			return err
		}

		cargos := make([]model.OrderCargo, len(req.CargoItems))
		for i, it := range req.CargoItems {
			cargos[i] = model.OrderCargo{
				OrderID:          &order.OrderID,
				CargoName:        &it.CargoName,
				CargoType:        &it.CargoType,
				Quantity:         &it.Quantity,
				WeightTon:        &it.WeightTon,
				VolumeCubicMeter: &it.VolumeCubM,
				UnitPrice:        &it.UnitPrice,
				Subtotal:         &it.Subtotal,
			}
		}
		if err := tx.Create(&cargos).Error; err != nil {
			return err
		}

		usages := make([]model.SegmentCapacityUsage, len(segments))
		for i, seg := range segments {
			usages[i] = model.SegmentCapacityUsage{
				OrderID:     &order.OrderID,
				LineID:      &req.LineID,
				VesselID:    &req.VesselID,
				VoyageDate:  voyageDateObj,
				StartPortID: &seg[0],
				EndPortID:   &seg[1],
				OccupiedTon: totalWeight,
			}
		}
		if err := tx.Create(&usages).Error; err != nil {
			return err
		}

		if err := s.voyageCargoNoteDAO.AddCumulativeCapacity(tx, loadNote.NoteID, totalWeight); err != nil {
			return err
		}
		if err := s.voyageCargoNoteDAO.AddCumulativeCapacity(tx, unloadNote.NoteID, totalWeight); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		logger.Error("create order failed", "error", err)
		return nil, err
	}

	cache.DeletePrefix("voyage_rec:")

	logger.Info("order created", "order_id", order.OrderID, "order_no", order.OrderNo, "calculated_cost", totalCost)
	return order, nil
}

func (s *orderServiceImpl) CancelOrder(ctx context.Context, orderID int64) error {
	logger := Logger.With("method", "CancelOrder", "order_id", orderID)
	logger.Info("cancelling order")

	ctx, cancel := WithTimeout(ctx)
	defer cancel()

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order model.ShippingOrder
		err := tx.Set("gorm:query_option", "FOR UPDATE").
			Preload("LoadNote").
			Preload("UnloadNote").
			First(&order, orderID).Error
		if err != nil {
			return pkgerr.NotFound("order not found")
		}
		if order.OrderStatus != nil && *order.OrderStatus == 4 {
			return pkgerr.Conflict("order already cancelled")
		}
		if order.LoadNote == nil || order.UnloadNote == nil {
			return pkgerr.NotFound("cargo note not found")
		}
		lineID := *order.LoadNote.LineID
		vesselID := *order.LoadNote.VesselID
		voyageDate := order.LoadNote.VoyageDate.Format("2006-01-02")

		lockName := VoyageLockKey(lineID, vesselID, voyageDate)
		ok, err := AcquireLock(tx, lockName, 10)
		if err != nil || !ok {
			return pkgerr.New(pkgerr.CodeTooManyRequests, "failed to acquire lock")
		}
		defer ReleaseLock(tx, lockName)

		if err := s.voyageCargoNoteDAO.AddCumulativeCapacity(tx, *order.LoadNoteID, -*order.TotalWeightTon); err != nil {
			return err
		}
		if err := s.voyageCargoNoteDAO.AddCumulativeCapacity(tx, *order.UnloadNoteID, -*order.TotalWeightTon); err != nil {
			return err
		}

		if err := tx.Model(&model.ShippingOrder{}).Where("order_id = ?", orderID).Update("delete_time", gorm.Expr("NOW()")).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.OrderCargo{}).Where("order_id = ?", orderID).Update("delete_time", gorm.Expr("NOW()")).Error; err != nil {
			return err
		}
		if err := tx.Where("order_id = ?", orderID).Delete(&model.SegmentCapacityUsage{}).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		logger.Error("cancel order failed", "error", err)
		return err
	}

	cache.DeletePrefix("voyage_rec:")
	logger.Info("order cancelled")
	return nil
}

func (s *orderServiceImpl) UpdateOrderStatus(ctx context.Context, orderID int64, newStatus int8) error {
	logger := Logger.With("method", "UpdateOrderStatus", "order_id", orderID, "new_status", newStatus)
	logger.Info("updating order status")

	order, err := s.orderDAO.GetByID(orderID)
	if err != nil {
		return pkgerr.NotFound("order not found")
	}
	oldStatus := int8(0)
	if order.OrderStatus != nil {
		oldStatus = *order.OrderStatus
	}
	if err := s.stateMachine.Transition(oldStatus, newStatus); err != nil {
		logger.Warn("invalid state transition", "from", oldStatus, "to", newStatus)
		return pkgerr.BadRequest("invalid state transition")
	}
	order.OrderStatus = &newStatus
	if err := s.orderDAO.Update(order); err != nil {
		logger.Error("update failed", "error", err)
		return err
	}

	// WebSocket push
	if order.ShipperCompanyID != nil {
		if err := s.wsSvc.PushOrderStatusUpdate(*order.ShipperCompanyID, "shipper", orderID, newStatus); err != nil {
			logger.Error("failed to push websocket notification", "error", err)
		}
	}

	logger.Info("order status updated")
	return nil
}

func (s *orderServiceImpl) GetOrderByID(ctx context.Context, orderID int64) (*model.ShippingOrder, error) {
	var order model.ShippingOrder
	err := s.db.Scopes(dao.NotDeleted).
		Preload("OrderCargos").
		Preload("LoadNote").
		Preload("UnloadNote").
		First(&order, orderID).Error
	if err != nil {
		return nil, pkgerr.NotFound("order not found")
	}
	return &order, nil
}

func (s *orderServiceImpl) ListOrdersByShipper(ctx context.Context, shipperCompanyID int64, req PageRequest) ([]model.ShippingOrder, int64, error) {
	if req.AllowedSort == nil {
		req.AllowedSort = DefaultOrderSortFields()
	}
	query := s.db.Model(&model.ShippingOrder{}).
		Scopes(dao.NotDeleted).
		Where("shipper_company_id = ?", shipperCompanyID)

	paginatedQuery, total, err := Paginate(query, req, &model.ShippingOrder{})
	if err != nil {
		return nil, 0, err
	}

	var orders []model.ShippingOrder
	if err := paginatedQuery.Find(&orders).Error; err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

// OrderTracking
type OrderTracking struct {
	OrderID          int64      `json:"order_id"`
	OrderNo          string     `json:"order_no"`
	OrderStatus      int8       `json:"order_status"`
	StatusName       string     `json:"status_name"`
	LoadTime         *time.Time `json:"load_time"`
	UnloadTime       *time.Time `json:"unload_time"`
	DeparturePort    string     `json:"departure_port"`
	DestinationPort  string     `json:"destination_port"`
	DeparturePlanned *time.Time `json:"departure_planned"`
	DepartureActual  *time.Time `json:"departure_actual"`
	ArrivalPlanned   *time.Time `json:"arrival_planned"`
	ArrivalActual    *time.Time `json:"arrival_actual"`
	VesselName       string     `json:"vessel_name"`
	LineName         string     `json:"line_name"`
}

func (s *orderServiceImpl) GetOrderTracking(ctx context.Context, orderID int64) (*OrderTracking, error) {
	var order model.ShippingOrder
	err := s.db.Scopes(dao.NotDeleted).
		Preload("LoadNote").
		Preload("UnloadNote").
		Preload("DeparturePort").
		Preload("DestinationPort").
		First(&order, orderID).Error
	if err != nil {
		return nil, pkgerr.NotFound("order not found")
	}

	if order.LoadNote == nil {
		return nil, pkgerr.NotFound("load note not found")
	}
	lineID := *order.LoadNote.LineID
	vesselID := *order.LoadNote.VesselID
	voyageDate := order.LoadNote.VoyageDate

	var vessel model.Vessel
	vesselName := ""
	if err := s.db.First(&vessel, vesselID).Error; err == nil {
		vesselName = vessel.VesselName
	}

	var line model.ShippingLine
	lineName := ""
	if err := s.db.First(&line, lineID).Error; err == nil {
		lineName = line.LineName
	}

	var departureBerthing, arrivalBerthing model.VoyageBerthing
	s.db.Where("line_id = ? AND vessel_id = ? AND voyage_date = ? AND port_id = ?",
		lineID, vesselID, voyageDate, *order.DeparturePortID).
		First(&departureBerthing)
	s.db.Where("line_id = ? AND vessel_id = ? AND voyage_date = ? AND port_id = ?",
		lineID, vesselID, voyageDate, *order.DestinationPortID).
		First(&arrivalBerthing)

	statusName := map[int8]string{
		0: "Draft", 1: "Confirmed", 2: "In Transit", 3: "Completed", 4: "Cancelled",
	}[*order.OrderStatus]

	tracking := &OrderTracking{
		OrderID:          order.OrderID,
		OrderNo:          order.OrderNo,
		OrderStatus:      *order.OrderStatus,
		StatusName:       statusName,
		LoadTime:         getNoteTime(order.LoadNote),
		UnloadTime:       getNoteTime(order.UnloadNote),
		DeparturePort:    getPortName(order.DeparturePort),
		DestinationPort:  getPortName(order.DestinationPort),
		DeparturePlanned: departureBerthing.PlannedArrivalTime,
		DepartureActual:  departureBerthing.ActualArrivalTime,
		ArrivalPlanned:   arrivalBerthing.PlannedArrivalTime,
		ArrivalActual:    arrivalBerthing.ActualArrivalTime,
		VesselName:       vesselName,
		LineName:         lineName,
	}
	return tracking, nil
}

func getNoteTime(note *model.VoyageCargoNote) *time.Time {
	if note == nil {
		return nil
	}
	return &note.CreateTime
}

func getPortName(port *model.Port) string {
	if port == nil {
		return ""
	}
	return port.PortName
}
