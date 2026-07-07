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
	"backend/pkg/timeutil"

	"gorm.io/gorm"
)

// OrderService 订单服务接口，包含创建、取消、状态更新、查询和追踪。
type OrderService interface {
	CreateOrder(ctx context.Context, req *CreateOrderRequest) (*model.ShippingOrder, error)
	CancelOrder(ctx context.Context, orderID int64) error
	UpdateOrderStatus(ctx context.Context, orderID int64, newStatus int8) error
	GetOrderByID(ctx context.Context, orderID int64) (*model.ShippingOrder, error)
	ListOrdersByShipper(ctx context.Context, shipperCompanyID int64, req PageRequest, orderNo string, orderStatus *int8) ([]model.ShippingOrder, int64, error)
	ListOrdersByShippingCompany(ctx context.Context, companyID int64, req PageRequest, orderNo string, orderStatus *int8) ([]model.ShippingOrder, int64, error)
	ListAllOrders(ctx context.Context, req PageRequest, orderNo string, orderStatus *int8) ([]model.ShippingOrder, int64, error)
	CheckOrderBelongsToShippingCompany(ctx context.Context, orderID, shippingCompanyID int64) (bool, error)
	CheckOrderBelongsToShipper(ctx context.Context, orderID, shipperCompanyID int64) (bool, error)
	GetOrderTracking(ctx context.Context, orderID int64) (*OrderTracking, error)
	PayOrder(ctx context.Context, orderID int64) error
}

// CreateOrderRequest 创建订单的请求参数。
type CreateOrderRequest struct {
	ShipperCompanyID   int64
	CityID             int64
	LineID             int64
	VesselID           int64
	VoyageDate         string
	StartPortID        int64
	EndPortID          int64
	CargoItems         []CargoItem
	ShipperContact     string
	ConsigneeContact   string
	ExpectedDeparture  *string
	ExpectedArrival    *string
	ShippingCompanyID  int64 // 仅 shipping 角色设置，用于校验 vessel/line 归属
}

// CargoItem 货物条目。
type CargoItem struct {
	CargoName    string  `json:"cargo_name"`
	CargoType    string  `json:"cargo_type"`
	Quantity     float64 `json:"quantity"`
	WeightTon    float64 `json:"weight_ton"`
	VolumeCubM   float64 `json:"volume_cub_m"`
	UnitPrice    float64 `json:"unit_price"`
	Subtotal     float64 `json:"subtotal"`
	UnloadPortID *int64  `json:"unload_port_id,omitempty"` // 卸货港ID，为空则使用订单的end_port_id
}

// orderServiceImpl 订单服务实现。
//
// 依赖的组件说明（按类别分组）：
//
// 【数据库】
//   - db:                *gorm.DB 实例。用于开启事务（Transaction）。
//     执行 SELECT FOR UPDATE 行锁、直接执行复杂查询。
//
// 【DAO 层—数据访问对象，每个对应一张表的 CRUD。】
//   - orderDAO:           shipping_order 表的增删改查。
//   - orderCargoDAO:      order_cargo 表的增删改查（订单的货物明细）。
//   - segmentUsageDAO:    segment_capacity_usage 表的增删改查。
//     记录每个订单在每个航段上占用的吨位。关键操作：
//     GetOccupiedTons() 获取某航段的已用容量（SUM 聚合）。
//   - voyageCargoNoteDAO: voyage_cargo_note 表的操作。
//     查找装卸货通知单（LOAD/UNLOAD），更新累积预订容量。
//     FindByPortAndOp() 查询某个港口上的装/卸货记录。
//     AddCumulativeCapacity() 更新累积已预订吨位。
//   - vesselDAO:          船舶表。用于获取船舶的 MaxDeadweightTon（最大载重吨），
//     运力计算的基础。
//   - shippingLineDAO:    航线表。用于获取航线距离（TotalDistanceNm）
//     和港口序列（PortSequence JSON）。
//
// 【Biz 层—纯业务逻辑计算，不依赖 I/O。】
//   - portSeqParser:     PortSequenceParser —— 将 JSON "[1,2,3]" 解析为 []int64。
//   - segCalc:           SegmentCalculator —— 根据港口序列和起止港，
//     计算途径的邻接航段列表（用于逐段校验容量）。
//   - capChecker:        CapacityChecker —— 对所有航段执行容量校验：
//     每个航段中 已占用 + 新货物 <= 船舶最大载重。
//     如果任何一个航段超容，拒绝下单。
//   - costCalc:          CostCalculator —— 汇总计算货物的总重量、总体积、总费用。
//     注：运费（元）= 总重量 x 总航程 x 基础费率 x 货种系数。
//     这部分在 service 层计算（因为涉及 config 中的费率配置），
//     不在 biz 层处理。
//   - orderNoGen:        OrderNoGenerator —— 生成唯一订单号。
//     格式：ORD + YYYYMMDD + 8位随机hex。
//   - stateMachine:      OrderStateMachine —— 订单状态转换规则校验。
//     可转换：0→1/4, 1→2/4, 2→3/4, 3/4 为终态不可转换。
//
// 【其他 Service】
//   - wsSvc:             WebSocketService —— 订单状态变更后，
//     通过 WebSocket 向货主推送实时通知。
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

// NewOrderService 创建订单服务实例。
//
// 构造函数参数说明：
//
// 【数据库连接】
//   db — 用于开启事务。创建订单涉及 5 张表的写入，需用 Transaction 保证原子性。
//
// 【DAO — 数据访问对象】
//   orderDAO           — shipping_order 表 CRUD
//   orderCargoDAO      — order_cargo 表 CRUD（订单的各项货物明细）
//   segmentUsageDAO    — segment_capacity_usage 表（航段容量占用，核心运力管理表）
//   voyageCargoNoteDAO — voyage_cargo_note 表（航次货物记录，用于校验装卸货计划和更新累积运力）
//   vesselDAO          — vessel 表（获取船舶最大载重 DWT）
//   shippingLineDAO    — shipping_line 表（获取航线距离和港口序列）
//
// 【Biz — 纯业务逻辑组件】
//   portSeqParser — JSON 港口序列解析器："[1,2,3]" → [1,2,3]。
//   segCalc       — 航段计算器：根据港口序列和起止港 → [(起·终), (终·止)]。
//   capChecker    — 容量校验器：检查所有航段上 已占+新增 <= 最大载重。
//   costCalc      — 成本计算器：汇总货物总重/总体积/总费用。
//   orderNoGen    — 订单号生成器：ORD + YYYYMMDD + 8位随机hex。
//   stateMachine  — 状态机：校验订单状态转换合法性（草稿→确认→运输中→完成/取消）。
//
// 【其他 Service】
//   wsSvc — WebSocket 推送服务。状态变更时向货主推送实时通知。
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

// CreateOrder 创建运输订单，包含完整业务逻辑：
//  1. 成本计算（货物汇总）
//  2. 校验船舶载重、航线港口序列
//  3. 计算运费（吨 × 海里 × 基础费率 × 货物系数）
//  4. 解析港口序列，计算途径航段
//  5. 校验装卸货 cargo note 存在
//  6. 事务内：
//     a. GET_LOCK 获取航次锁（防止并发超售）
//     b. SELECT FOR UPDATE 锁定舱位记录
//     c. 容量检查（所有航段剩余容量 >= 货物重量）
//     d. 创建订单、货物条目、航段占用记录
//     e. 更新 cargo note 累积容量
//  7. 清除运力推荐缓存
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

	// 如果调用方是船公司角色，校验船舶和航线是否属于该公司
	if req.ShippingCompanyID > 0 {
		if vessel.ShippingCompanyID == nil || *vessel.ShippingCompanyID != req.ShippingCompanyID {
			logger.Error("vessel does not belong to shipping company", "vessel_id", req.VesselID, "company_id", req.ShippingCompanyID)
			return nil, pkgerr.Forbidden("vessel does not belong to your company")
		}
	}

	maxWeight := *vessel.MaxDeadweightTon

	line, err := s.shippingLineDAO.GetByID(req.LineID)
	if err != nil || line.PortSequence == nil {
		logger.Error("shipping line not found", "line_id", req.LineID)
		return nil, pkgerr.NotFound("shipping line not found or port sequence missing")
	}

	// 校验航线归属
	if req.ShippingCompanyID > 0 {
		if line.ShippingCompanyID == nil || *line.ShippingCompanyID != req.ShippingCompanyID {
			logger.Error("line does not belong to shipping company", "line_id", req.LineID, "company_id", req.ShippingCompanyID)
			return nil, pkgerr.Forbidden("shipping line does not belong to your company")
		}
	}

	// 解析港口序列
	portIDs, err := s.portSeqParser.Parse(*line.PortSequence)
	if err != nil {
		logger.Error("parse port sequence failed", "error", err)
		return nil, pkgerr.BadRequest("invalid port sequence")
	}

	// 确定每个货物的卸货港，未指定则使用订单的目的港
	type cargoUnload struct {
		item     CargoItem
		unloadID int64
	}
	cargoUnloads := make([]cargoUnload, len(req.CargoItems))
	farthestIdx := -1
	for i, c := range req.CargoItems {
		unloadID := c.UnloadPortID
		if unloadID == nil {
			unloadID = &req.EndPortID
		}
		ui := indexOf(portIDs, *unloadID)
		si := indexOf(portIDs, req.StartPortID)
		ei := indexOf(portIDs, req.EndPortID)
		if si == -1 { return nil, pkgerr.BadRequest("start port not in route") }
		if ei == -1 { return nil, pkgerr.BadRequest("end port not in route") }
		if ui == -1 { return nil, pkgerr.BadRequest("unload port not in route") }
		if ui < si { return nil, pkgerr.BadRequest("unload port is before start port") }
		if ui > ei { return nil, pkgerr.BadRequest("unload port is beyond order end port") }
		cargoUnloads[i] = cargoUnload{item: c, unloadID: *unloadID}
		if ui > farthestIdx { farthestIdx = ui }
	}

	// 航线总里程（用于运费比例计算）
	totalDistance := 0.0
	if line.TotalDistanceNm != nil { totalDistance = *line.TotalDistanceNm }

	// 计算有效航段（从起运港到最远卸货港）
	effectiveEnd := portIDs[farthestIdx]
	segments, err := s.segCalc.Calculate(portIDs, req.StartPortID, effectiveEnd)
	if err != nil {
		if errors.Is(err, biz.ErrPortNotFoundInSeq) || errors.Is(err, biz.ErrStartAfterEnd) {
			return nil, pkgerr.BadRequest(err.Error())
		}
		return nil, err
	}

	// 每个航段上的实际占用重量 = 所有在此航段之后（含）卸货的货物重量之和
	segWeight := make(map[[2]int64]float64)
	for _, seg := range segments {
		var w float64
		for _, cu := range cargoUnloads {
			ui := indexOf(portIDs, cu.unloadID)
			ei := indexOf(portIDs, seg[1])
			if ui >= ei { w += cu.item.WeightTon }
		}
		segWeight[seg] = w
	}

	// 先查找 cargo note，以便获取自定义单价
	loadNote, err := s.voyageCargoNoteDAO.FindByPortAndOp(req.LineID, req.VesselID, req.VoyageDate, req.StartPortID, "LOAD")
	if err != nil {
		logger.Warn("load note not found, auto-creating", "port", req.StartPortID)
		var berthing model.VoyageBerthing
		if berthErr := s.db.Where("line_id = ? AND vessel_id = ? AND voyage_date = ? AND port_id = ?",
			req.LineID, req.VesselID, req.VoyageDate, req.StartPortID).First(&berthing).Error; berthErr != nil {
			return nil, pkgerr.NotFound("no voyage schedule for start port")
		}
		loadNote = &model.VoyageCargoNote{
			LineID:        &req.LineID,
			VesselID:      &req.VesselID,
			VoyageDate:    MustParseDate(req.VoyageDate),
			SequenceNo:    berthing.SequenceNo,
			OperationType: strPtr("LOAD"),
			CargoName:     strPtr("General Cargo"),
			CargoType:     strPtr("bulk"),
		}
		if err := s.voyageCargoNoteDAO.Create(loadNote); err != nil {
			return nil, pkgerr.Internal("failed to create load note")
		}
	}
	// 分段计算运费
	totalCost := 0.0
	cfg := config.Get()
	baseRate := cfg.Freight.BaseRatePerTonNm
	// 如果 cargo note 设置了自定义单价，优先使用
	customRate := 0.0
	if loadNote != nil && loadNote.UnitPrice != nil && *loadNote.UnitPrice > 0 {
		customRate = *loadNote.UnitPrice
	}
	for _, cu := range cargoUnloads {
		ci := indexOf(portIDs, req.StartPortID)
		ui := indexOf(portIDs, cu.unloadID)
		distance := totalDistance * float64(ui-ci) / float64(len(portIDs)-1)
		cargoFactor := 1.0
		if f, ok := cfg.Freight.CargoTypeFactors[cu.item.CargoType]; ok { cargoFactor = f }
		rate := baseRate * cargoFactor
		if customRate > 0 {
			rate = customRate
		}
		totalCost += cu.item.WeightTon * distance * rate
	}

	unloadNote, err := s.voyageCargoNoteDAO.FindByPortAndOp(req.LineID, req.VesselID, req.VoyageDate, req.EndPortID, "UNLOAD")
	if err != nil {
		logger.Warn("unload note not found, auto-creating", "port", req.EndPortID)
		var berthing model.VoyageBerthing
		if berthErr := s.db.Where("line_id = ? AND vessel_id = ? AND voyage_date = ? AND port_id = ?",
			req.LineID, req.VesselID, req.VoyageDate, req.EndPortID).First(&berthing).Error; berthErr != nil {
			return nil, pkgerr.NotFound("no voyage schedule for end port")
		}
		unloadNote = &model.VoyageCargoNote{
			LineID:        &req.LineID,
			VesselID:      &req.VesselID,
			VoyageDate:    MustParseDate(req.VoyageDate),
			SequenceNo:    berthing.SequenceNo,
			OperationType: strPtr("UNLOAD"),
			CargoName:     strPtr("General Cargo"),
			CargoType:     strPtr("bulk"),
		}
		if err := s.voyageCargoNoteDAO.Create(unloadNote); err != nil {
			return nil, pkgerr.Internal("failed to create unload note")
		}
	}

	var order *model.ShippingOrder
	voyageDateObj := MustParseDate(req.VoyageDate)
	lockName := VoyageLockKey(req.LineID, req.VesselID, req.VoyageDate)

	// ── 开启数据库事务 ──
	// 涉及 4 张表的写入（shipping_order, order_cargo, segment_capacity_usage,
	// voyage_cargo_note 的更新），必须保证原子性：要么全部成功，要么全部失败。
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		// 1. GET_LOCK：获取航次级互斥锁，防止并发超卖
		// 锁名格式 "voyage_{lineID}_{vesselID}_{date}"，只锁当前航次，不影响其他航次
		ok, err := AcquireLock(tx, lockName, 10)
		if err != nil || !ok {
			return pkgerr.New(pkgerr.CodeTooManyRequests, "failed to acquire lock, please retry")
		}
		defer ReleaseLock(tx, lockName)

		// 2. SELECT FOR UPDATE：锁定每个航段的已有容量记录
		// 防止在当前事务提交前，另一个事务读到旧的容量数据
		// 如果记录不存在（首次下单），不报错（gorm.ErrRecordNotFound），
		// 因为后续创建时会自动 insert
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

		// 3. 容量校验：逐段检查 已占吨位 + 本段货物重量 <= 船舶最大载重
		// 与单段订单不同，多卸货港时每段只算实际经过该段的货物
		for _, seg := range segments {
			used, err := s.segmentUsageDAO.GetOccupiedTons(req.LineID, req.VesselID, req.VoyageDate, seg[0], seg[1])
			if err != nil { return err }
			segW := segWeight[seg]
			remaining := maxWeight - used - segW
			if remaining < 0 {
				return pkgerr.New(pkgerr.CodeConflict, fmt.Sprintf("insufficient capacity at segment %d→%d, remaining: %.2f", seg[0], seg[1], remaining))
			}
		}

		// 4. 生成唯一订单号（ORD + 日期 + 8位随机hex）
		orderNo := s.orderNoGen.Generate()

		// 5. 解析可选日期参数
		var expDep, expArr *time.Time
		if req.ExpectedDeparture != nil {
			t, err := timeutil.ParseDate(*req.ExpectedDeparture)
			if err == nil { expDep = &t }
		}
		if req.ExpectedArrival != nil {
			t, err := timeutil.ParseDate(*req.ExpectedArrival)
			if err == nil { expArr = &t }
		}

		// 6. 创建订单主记录（shipping_order 表）
		// 注意：所有外键字段（shipper_company_id, city_id 等）都存为 *int64，
		// 因为在 GORM 模型中定义为 *int64（可空），必须取指针赋值
		order = &model.ShippingOrder{
			OrderNo:               orderNo,
			ShipperCompanyID:      &req.ShipperCompanyID,
			CityID:                &req.CityID,
			LoadNoteID:            &loadNote.NoteID,
			UnloadNoteID:          &unloadNote.NoteID,
			DeparturePortID:       &req.StartPortID,
			DestinationPortID:     &req.EndPortID,
			ExpectedDepartureDate: expDep,
			ExpectedArrivalDate:   expArr,
			TotalWeightTon:        &totalWeight,
			TotalVolumeCubicMeter: &totalVolume,
			TotalCost:             &totalCost,
			ShipperContact:        &req.ShipperContact,
			ConsigneeContact:      &req.ConsigneeContact,
			PaymentStatus:         PtrInt8(0),
			OrderStatus:           PtrInt8(0), // 0=待确认，海运公司审核后更新为1=已确认
		}
		if err := tx.Create(order).Error; err != nil {
			return err
		}

		// 7. 批量创建货物明细（order_cargo 表）
		// 每个 CargoItem 对应一条 order_cargo 记录
		// OrderID 在 order.Create 后由 GORM 自动填充
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

		// 8. 批量写入航段容量占用（segment_capacity_usage 表）
		// 每个航段一条记录，占用吨位按实际经过该段的货物重量计算
		usages := make([]model.SegmentCapacityUsage, len(segments))
		for i, seg := range segments {
			usages[i] = model.SegmentCapacityUsage{
				OrderID:     &order.OrderID,
				LineID:      &req.LineID,
				VesselID:    &req.VesselID,
				VoyageDate:  voyageDateObj,
				StartPortID: &seg[0],
				EndPortID:   &seg[1],
				OccupiedTon: segWeight[seg],
			}
		}
		if err := tx.Create(&usages).Error; err != nil {
			return err
		}

		// 9. 更新 LOAD/UNLOAD 通知单的累积预订容量
		// 每次创建订单时累加，取消订单时减去
		// 这个字段用于快速查询"这条航次已经订了多少吨"
		if err := s.voyageCargoNoteDAO.AddCumulativeCapacity(tx, loadNote.NoteID, totalWeight); err != nil {
			return err
		}
		if err := s.voyageCargoNoteDAO.AddCumulativeCapacity(tx, unloadNote.NoteID, totalWeight); err != nil {
			return err
		}

		return nil // 事务正常提交
	})

	if err != nil {
		logger.Error("create order failed", "error", err)
		return nil, err
	}

	// 10. 清除航次推荐缓存
	// 新订单创建后运力变化，旧的推荐结果不再准确
	// 下次查询时会重新计算并写入新缓存
	cache.DeletePrefix("voyage_rec:")

	logger.Info("order created", "order_id", order.OrderID, "order_no", order.OrderNo, "calculated_cost", totalCost)
	return order, nil
}

// CancelOrder 取消订单。事务内：释放累积运力 + 软删除订单/货物 + 物理删除容量占用
func (s *orderServiceImpl) CancelOrder(ctx context.Context, orderID int64) error {
	logger := Logger.With("method", "CancelOrder", "order_id", orderID)
	logger.Info("cancelling order")

	ctx, cancel := WithTimeout(ctx)
	defer cancel()

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		// 1. SELECT FOR UPDATE 锁定订单行，防止并发操作
		// 同时预加载 LoadNote 和 UnloadNote，获取 lineID/vesselID/date 用于解锁
		var order model.ShippingOrder
		err := tx.Set("gorm:query_option", "FOR UPDATE").First(&order, orderID).Error
		if err != nil {
			return pkgerr.NotFound("order not found")
		}
		// 再加载关联表（只读，无需锁定）
		if err := tx.Preload("LoadNote").Preload("UnloadNote").First(&order, orderID).Error; err != nil {
			return pkgerr.NotFound("order not found")
		}

		// 2. 校验订单状态：已取消的订单不可重复取消
		if order.OrderStatus != nil && *order.OrderStatus == 4 {
			return pkgerr.Conflict("order already cancelled")
		}
		if order.LoadNote == nil || order.UnloadNote == nil {
			return pkgerr.NotFound("cargo note not found")
		}

		// 3. 获取航次锁（与创建订单同一套锁机制），防止正在创建订单的同时取消
		lineID := *order.LoadNote.LineID
		vesselID := *order.LoadNote.VesselID
		voyageDate := order.LoadNote.VoyageDate.Format("2006-01-02")
		lockName := VoyageLockKey(lineID, vesselID, voyageDate)
		ok, err := AcquireLock(tx, lockName, 10)
		if err != nil || !ok {
			return pkgerr.New(pkgerr.CodeTooManyRequests, "failed to acquire lock")
		}
		defer ReleaseLock(tx, lockName)

		// 4. 释放 LOAD/UNLOAD 通知单的累积容量（通过负值累加）
		// 例如之前 +500，现在 +(-500)，回到 0
		if err := s.voyageCargoNoteDAO.AddCumulativeCapacity(tx, *order.LoadNoteID, -*order.TotalWeightTon); err != nil {
			return err
		}
		if err := s.voyageCargoNoteDAO.AddCumulativeCapacity(tx, *order.UnloadNoteID, -*order.TotalWeightTon); err != nil {
			return err
		}

		// 5. 清除本订单的运力占用记录
		// 订单表和货物表：软删除（设置 delete_time，数据仍保留用于审计）
		// 容量占用表：物理删除（因为不需要保留历史运力记录）
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

	// 取消成功后清除推荐缓存，释放的运力可以在下次推荐中体现
	cache.DeletePrefix("voyage_rec:")
	logger.Info("order cancelled")
	return nil
}

// UpdateOrderStatus 更新订单状态。校验：状态机合法性 + DAO 持久化 + WebSocket 推送
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

	// 状态更新后，通过 WebSocket 向该订单的货主推送实时通知
	// 这样货主在不刷新页面的情况下就能看到状态变化
	if order.ShipperCompanyID != nil {
		if err := s.wsSvc.PushOrderStatusUpdate(*order.ShipperCompanyID, "shipper", orderID, newStatus); err != nil {
			logger.Error("failed to push websocket notification", "error", err)
		}
	}

	logger.Info("order status updated")
	return nil
}

// GetOrderByID 查询订单详情。通过 GORM Preload 联表加载关联对象
func (s *orderServiceImpl) GetOrderByID(ctx context.Context, orderID int64) (*model.ShippingOrder, error) {
	var order model.ShippingOrder
	err := s.db.Scopes(dao.NotDeleted).
		Preload("ShipperCompany").
		Preload("City").
		Preload("OrderCargos").
		Preload("LoadNote").
		Preload("UnloadNote").
		Preload("DeparturePort").
		Preload("DestinationPort").
		First(&order, orderID).Error
	if err != nil {
		return nil, pkgerr.NotFound("order not found")
	}
	return &order, nil
}

// ListOrdersByShipper 分页查询订单列表。步骤：组装 query → Paginate 排序分页 → Preload City → Find
func (s *orderServiceImpl) ListOrdersByShipper(ctx context.Context, shipperCompanyID int64, req PageRequest, orderNo string, orderStatus *int8) ([]model.ShippingOrder, int64, error) {
	if req.AllowedSort == nil {
		req.AllowedSort = DefaultOrderSortFields()
	}
	query := s.db.Model(&model.ShippingOrder{}).
		Scopes(dao.NotDeleted).
		Where("shipper_company_id = ?", shipperCompanyID)

	if orderNo != "" {
		query = query.Where("order_no LIKE ?", "%"+orderNo+"%")
	}
	if orderStatus != nil {
		query = query.Where("order_status = ?", *orderStatus)
	}

	paginatedQuery, total, err := Paginate(query, req, &model.ShippingOrder{})
	if err != nil {
		return nil, 0, err
	}

	var orders []model.ShippingOrder
	if err := paginatedQuery.
		Preload("ShipperCompany").
		Preload("City").
		Preload("DeparturePort").
		Preload("DestinationPort").
		Find(&orders).Error; err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

// ListOrdersByShippingCompany 分页查询指定船公司的订单列表。
// 通过 voyage_cargo_note → shipping_line 关联找到属于该船公司的订单。
func (s *orderServiceImpl) ListOrdersByShippingCompany(ctx context.Context, companyID int64, req PageRequest, orderNo string, orderStatus *int8) ([]model.ShippingOrder, int64, error) {
	if req.AllowedSort == nil {
		req.AllowedSort = DefaultOrderSortFields()
	}
	query := s.db.Model(&model.ShippingOrder{}).Scopes(dao.NotDeleted).
		Joins("JOIN voyage_cargo_note ON shipping_order.load_note_id = voyage_cargo_note.note_id").
		Joins("JOIN shipping_line ON voyage_cargo_note.line_id = shipping_line.line_id").
		Where("shipping_line.shipping_company_id = ? AND shipping_line.delete_time IS NULL", companyID)

	if orderNo != "" {
		query = query.Where("shipping_order.order_no LIKE ?", "%"+orderNo+"%")
	}
	if orderStatus != nil {
		query = query.Where("shipping_order.order_status = ?", *orderStatus)
	}

	paginatedQuery, total, err := Paginate(query, req, &model.ShippingOrder{})
	if err != nil {
		return nil, 0, err
	}

	var orders []model.ShippingOrder
	if err := paginatedQuery.
		Preload("City").
		Preload("DeparturePort").
		Preload("DestinationPort").
		Preload("ShipperCompany").
		Find(&orders).Error; err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

// ListAllOrders 分页查询所有订单（admin 角色使用，不限制 shipper_company_id）。
func (s *orderServiceImpl) ListAllOrders(ctx context.Context, req PageRequest, orderNo string, orderStatus *int8) ([]model.ShippingOrder, int64, error) {
	if req.AllowedSort == nil {
		req.AllowedSort = DefaultOrderSortFields()
	}
	query := s.db.Model(&model.ShippingOrder{}).Scopes(dao.NotDeleted)
	if orderNo != "" {
		query = query.Where("order_no LIKE ?", "%"+orderNo+"%")
	}
	if orderStatus != nil {
		query = query.Where("order_status = ?", *orderStatus)
	}
	paginatedQuery, total, err := Paginate(query, req, &model.ShippingOrder{})
	if err != nil {
		return nil, 0, err
	}
	var orders []model.ShippingOrder
	if err := paginatedQuery.
		Preload("ShipperCompany").
		Preload("DeparturePort").
		Preload("DestinationPort").
		Find(&orders).Error; err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

// CheckOrderBelongsToShippingCompany 检查指定订单是否属于某船公司。
// 通过 load_note_id → voyage_cargo_note.line_id → shipping_line.shipping_company_id 链路判断。
func (s *orderServiceImpl) CheckOrderBelongsToShippingCompany(ctx context.Context, orderID, shippingCompanyID int64) (bool, error) {
	ctx, cancel := WithTimeout(ctx)
	defer cancel()

	var count int64
	err := s.db.Model(&model.ShippingOrder{}).
		Scopes(dao.NotDeleted).
		Joins("JOIN voyage_cargo_note ON shipping_order.load_note_id = voyage_cargo_note.note_id").
		Joins("JOIN shipping_line ON voyage_cargo_note.line_id = shipping_line.line_id").
		Where("shipping_order.order_id = ? AND shipping_line.shipping_company_id = ? AND shipping_line.delete_time IS NULL", orderID, shippingCompanyID).
		Count(&count).Error
	return count > 0, err
}

// CheckOrderBelongsToShipper 检查指定订单是否属于某货主。
func (s *orderServiceImpl) CheckOrderBelongsToShipper(ctx context.Context, orderID, shipperCompanyID int64) (bool, error) {
	var count int64
	err := s.db.Model(&model.ShippingOrder{}).Scopes(dao.NotDeleted).
		Where("order_id = ? AND shipper_company_id = ?", orderID, shipperCompanyID).
		Count(&count).Error
	return count > 0, err
}

// OrderTracking 订单追踪信息结构体。包含订单状态 + 装卸货时间 + 起止港靠泊计划/实际时间 + 船舶航线名称
type OrderTracking struct {
	OrderID              int64      `json:"order_id"`
	OrderNo              string     `json:"order_no"`
	OrderStatus          int8       `json:"order_status"`
	StatusName           string     `json:"status_name"`
	LoadTime             *time.Time `json:"load_time"`
	UnloadTime           *time.Time `json:"unload_time"`
	DeparturePort        string     `json:"departure_port"`
	DestinationPort      string     `json:"destination_port"`
	ExpectedDeparture    *time.Time `json:"expected_departure"`
	ExpectedArrival      *time.Time `json:"expected_arrival"`
	DepartureBerthingID  int64      `json:"departure_berthing_id"`
	ArrivalBerthingID    int64      `json:"arrival_berthing_id"`
	DeparturePlanned     *time.Time `json:"departure_planned"`
	DepartureActual      *time.Time `json:"departure_actual"`
	ArrivalPlanned       *time.Time `json:"arrival_planned"`
	ArrivalActual        *time.Time `json:"arrival_actual"`
	VesselName           string     `json:"vessel_name"`
	LineName             string     `json:"line_name"`
}

// PayOrder 虚拟支付——将订单支付状态更新为已支付。
func (s *orderServiceImpl) PayOrder(ctx context.Context, orderID int64) error {
	logger := Logger.With("method", "PayOrder", "order_id", orderID)
	var order model.ShippingOrder
	if err := s.db.Scopes(dao.NotDeleted).First(&order, orderID).Error; err != nil {
		return pkgerr.NotFound("order not found")
	}
	order.PaymentStatus = PtrInt8(1)
	if err := s.orderDAO.Update(&order); err != nil {
		logger.Error("failed to update payment status", "error", err)
		return err
	}
	logger.Info("order paid", "order_id", orderID)
	return nil
}

// GetOrderTracking 查询物流追踪信息。联表查询：先查订单+关联表，再分别查船舶名/航线名/靠泊记录
func (s *orderServiceImpl) GetOrderTracking(ctx context.Context, orderID int64) (*OrderTracking, error) {
	logger := Logger.With("method", "GetOrderTracking", "order_id", orderID)
	// 1. 查订单主表 + 预加载城市/装卸货单/起止港（GORM 关联查询）
	var order model.ShippingOrder
	err := s.db.Scopes(dao.NotDeleted).
		Preload("City").
		Preload("LoadNote").
		Preload("UnloadNote").
		Preload("DeparturePort").
		Preload("DestinationPort").
		First(&order, orderID).Error
	if err != nil {
		return nil, pkgerr.NotFound("order not found")
	}

	// 2. 从 LoadNote 中获取 lineID/vesselID/voyageDate（追踪信息的核心三元组）
	if order.LoadNote == nil {
		return nil, pkgerr.NotFound("load note not found")
	}
	lineID := *order.LoadNote.LineID
	vesselID := *order.LoadNote.VesselID
	voyageDate := order.LoadNote.VoyageDate

	// 3. 分别查船舶名称和航线名称（单独查询而非 Preload，因为 Order 模型没有直接关联）
	var vessel model.Vessel
	vesselName := ""
	if err := s.db.Scopes(dao.NotDeleted).First(&vessel, vesselID).Error; err == nil {
		vesselName = vessel.VesselName
	}

	var line model.ShippingLine
	lineName := ""
	if err := s.db.Scopes(dao.NotDeleted).First(&line, lineID).Error; err == nil {
		lineName = line.LineName
	}

	// 4. 查询 voyage_berthing 表获取起止港的靠泊计划时间和实际时间
	// 用于前端展示：计划几点到港、实际出发时间等
	// 找不到靠泊记录不报错，前端展示空值
	var departureBerthing, arrivalBerthing model.VoyageBerthing
	if err := s.db.Where("line_id = ? AND vessel_id = ? AND voyage_date = ? AND port_id = ?",
		lineID, vesselID, voyageDate, *order.DeparturePortID).
		First(&departureBerthing).Error; err != nil {
		logger.Warn("departure berthing not found", "error", err)
	}
	if err := s.db.Where("line_id = ? AND vessel_id = ? AND voyage_date = ? AND port_id = ?",
		lineID, vesselID, voyageDate, *order.DestinationPortID).
		First(&arrivalBerthing).Error; err != nil {
		logger.Warn("arrival berthing not found", "error", err)
	}

	if order.OrderStatus == nil {
		return nil, pkgerr.Internal("order status is nil")
	}
	statusName := map[int8]string{
		0: "Draft", 1: "Confirmed", 2: "In Transit", 3: "Completed", 4: "Cancelled",
	}[*order.OrderStatus]

	tracking := &OrderTracking{
		OrderID:             order.OrderID,
		OrderNo:             order.OrderNo,
		OrderStatus:         *order.OrderStatus,
		StatusName:          statusName,
		LoadTime:            getNoteTime(order.LoadNote),
		UnloadTime:          getNoteTime(order.UnloadNote),
		DeparturePort:       getPortName(order.DeparturePort),
		DestinationPort:     getPortName(order.DestinationPort),
		ExpectedDeparture:   order.ExpectedDepartureDate,
		ExpectedArrival:     order.ExpectedArrivalDate,
		DepartureBerthingID: departureBerthing.BerthingID,
		ArrivalBerthingID:   arrivalBerthing.BerthingID,
		DeparturePlanned:    departureBerthing.PlannedArrivalTime,
		DepartureActual:     departureBerthing.ActualArrivalTime,
		ArrivalPlanned:      arrivalBerthing.PlannedArrivalTime,
		ArrivalActual:       arrivalBerthing.ActualArrivalTime,
		VesselName:          vesselName,
		LineName:            lineName,
	}
	return tracking, nil
}

// getNoteTime 获取装卸货通知单的创建时间
func getNoteTime(note *model.VoyageCargoNote) *time.Time {
	if note == nil { return nil }
	return &note.CreateTime
}

// getPortName 获取港口名称
func getPortName(port *model.Port) string {
	if port == nil { return "" }
	return port.PortName
}

// indexOf 返回 portID 在切片中的下标，未找到返回 -1
func indexOf(ids []int64, target int64) int {
	for i, id := range ids {
		if id == target { return i }
	}
	return -1
}

