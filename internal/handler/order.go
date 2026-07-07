package handler

import (
	"encoding/json"
	"strconv"
	"time"

	"backend/internal/service"
	"backend/pkg/errors"
	"backend/pkg/response"
	"backend/pkg/validator"

	"github.com/gin-gonic/gin"
)

// getUserID 从 gin.Context 中安全获取 int64 类型的 user_id。
func getUserID(c *gin.Context) (int64, bool) {
	uid, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	switch v := uid.(type) {
	case int64:
		return v, true
	case float64:
		return int64(v), true
	case json.Number:
		n, _ := v.Int64()
		return n, true
	default:
		return 0, false
	}
}

// getShippingCompanyID 从 JWT 上下文中提取船公司 ID。
// 仅当 role == "shipping" 时返回有效值，否则返回 0。
func getShippingCompanyID(c *gin.Context) (int64, bool) {
	role, _ := c.Get("role")
	if role != "shipping" {
		return 0, false
	}
	uid, ok := c.Get("user_id")
	if !ok {
		return 0, false
	}
	id, ok := uid.(int64)
	return id, ok
}

// OrderHandler 处理订单相关请求。
//
// 订单是系统的核心业务对象，handler 层覆盖了完整的 CRUD 操作：
// 创建、查询单个、列表、取消、状态更新、追踪。
type OrderHandler struct {
	svc service.OrderService
}

// NewOrderHandler 创建订单处理器
func NewOrderHandler(svc service.OrderService) *OrderHandler {
	return &OrderHandler{svc: svc}
}

// createOrderRequest 创建订单的请求体。
//
// 字段说明：
//   - shipper_company_id: 货主 ID。shipper 角色必须等于 JWT 中的 user_id。
//   - city_id: 城市 ID（货主所在城市）。
//   - line_id: 航线 ID（决定港口序列和总航程）。
//   - vessel_id: 船舶 ID（决定最大载重）。
//   - voyage_date: 航次日期，格式 yyyy-MM-dd。
//   - start_port_id / end_port_id: 起止港口 ID，必须在 line 的 port_sequence 中。
//   - cargo_items: 货物列表。至少 1 项，根据 cargo_type 计算运费系数。
//   - expected_departure/arrival: 可选，预计离港/到港日期。
type createOrderRequest struct {
	ShipperCompanyID  int64               `json:"shipper_company_id" validate:"required"`
	CityID            int64               `json:"city_id" validate:"required"`
	LineID            int64               `json:"line_id" validate:"required"`
	VesselID          int64               `json:"vessel_id" validate:"required"`
	VoyageDate        string              `json:"voyage_date" validate:"required,date"`
	StartPortID       int64               `json:"start_port_id" validate:"required"`
	EndPortID         int64               `json:"end_port_id" validate:"required"`
	CargoItems        []service.CargoItem `json:"cargo_items" validate:"required,min=1,dive"`
	ShipperContact    string              `json:"shipper_contact"`
	ConsigneeContact  string              `json:"consignee_contact"`
	ExpectedDeparture *string             `json:"expected_departure,omitempty"`
	ExpectedArrival   *string             `json:"expected_arrival,omitempty"`
}

// CreateOrder 创建运输订单。
//
// 权限校验：
//   - shipper 角色：user_id 必须等于请求中的 shipper_company_id。
//   - shipping 角色：只能使用自己的船舶和航线（line_id / vessel_id 的所属公司必须等于 user_id）。
//   - admin 角色：可以为任意货主下单（不受限制）。
//
// 创建成功后，后端会执行完整的业务逻辑（运费计算、运力校验、容量占用、
// 事务提交），并清除航次推荐缓存。具体流程见 service.orderServiceImpl.CreateOrder。
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c.Writer, "invalid request body: "+err.Error())
		return
	}
	if err := validator.Validate(req); err != nil {
		response.BadRequest(c.Writer, err.Error())
		return
	}

	role, _ := c.Get("role")
	if role == "shipper" {
		if uid, ok := getUserID(c); !ok || uid != req.ShipperCompanyID {
			response.ErrorWithCode(c.Writer, errors.CodeForbidden, "shipper_company_id mismatch")
			return
		}
	}

	var shippingCompanyID int64
	if role == "shipping" {
		if uid, ok := getUserID(c); ok {
			shippingCompanyID = uid
		}
	}

	orderReq := &service.CreateOrderRequest{
		ShipperCompanyID:  req.ShipperCompanyID,
		CityID:            req.CityID,
		LineID:            req.LineID,
		VesselID:          req.VesselID,
		VoyageDate:        req.VoyageDate,
		StartPortID:       req.StartPortID,
		EndPortID:         req.EndPortID,
		CargoItems:        req.CargoItems,
		ShipperContact:    req.ShipperContact,
		ConsigneeContact:  req.ConsigneeContact,
		ExpectedDeparture: req.ExpectedDeparture,
		ExpectedArrival:   req.ExpectedArrival,
		ShippingCompanyID: shippingCompanyID,
	}

	order, err := h.svc.CreateOrder(c.Request.Context(), orderReq)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.ErrorWithCode(c.Writer, appErr.Code, appErr.Message)
			return
		}
		response.InternalServerError(c.Writer, "failed to create order")
		return
	}

	response.Success(c.Writer, order)
}

// GetOrder 根据 ID 查询订单详情。
// 返回的订单数据包含城市、货物明细、装卸货单和起止港信息（通过 GORM Preload）。
// 权限：shipping 角色只能查自己公司的订单。
func (h *OrderHandler) GetOrder(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid order id")
		return
	}

	if companyID, ok := getShippingCompanyID(c); ok {
		belongs, err := h.svc.CheckOrderBelongsToShippingCompany(c.Request.Context(), id, companyID)
		if err != nil || !belongs {
			response.ErrorWithCode(c.Writer, errors.CodeNotFound, "order not found")
			return
		}
	}

	if role, _ := c.Get("role"); role == "shipper" {
		if uid, ok := c.Get("user_id"); ok {
			if shipperID, ok2 := uid.(int64); ok2 {
				belongs, err := h.svc.CheckOrderBelongsToShipper(c.Request.Context(), id, shipperID)
				if err != nil || !belongs {
					response.ErrorWithCode(c.Writer, errors.CodeNotFound, "order not found")
					return
				}
			}
		}
	}

	order, err := h.svc.GetOrderByID(c.Request.Context(), id)
	if err != nil {
		response.ErrorWithCode(c.Writer, errors.CodeNotFound, "order not found")
		return
	}
	response.Success(c.Writer, order)
}

// CancelOrder 取消指定的订单。
//
// 后端操作：软删除订单及货物，释放航段运力占用，清除推荐缓存。
// 权限：shipping 角色只能取消自己公司的订单。
func (h *OrderHandler) CancelOrder(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid order id")
		return
	}

	if companyID, ok := getShippingCompanyID(c); ok {
		belongs, err := h.svc.CheckOrderBelongsToShippingCompany(c.Request.Context(), id, companyID)
		if err != nil || !belongs {
			response.ErrorWithCode(c.Writer, errors.CodeNotFound, "order not found")
			return
		}
	}

	if role, _ := c.Get("role"); role == "shipper" {
		if uid, ok := c.Get("user_id"); ok {
			if shipperID, ok2 := uid.(int64); ok2 {
				belongs, err := h.svc.CheckOrderBelongsToShipper(c.Request.Context(), id, shipperID)
				if err != nil || !belongs {
					response.ErrorWithCode(c.Writer, errors.CodeNotFound, "order not found")
					return
				}
			}
		}
	}

	err = h.svc.CancelOrder(c.Request.Context(), id)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.ErrorWithCode(c.Writer, appErr.Code, appErr.Message)
			return
		}
		response.InternalServerError(c.Writer, "failed to cancel order")
		return
	}
	response.Success(c.Writer, gin.H{"message": "order cancelled"})
}

// updateOrderStatusRequest 更新订单状态的请求体。
// Status 取值范围：0(草稿) 1(已确认) 2(运输中) 3(已完成) 4(已取消)
type updateOrderStatusRequest struct {
	Status     int8   `json:"status" validate:"required,min=0,max=4"`
	PortID     int64  `json:"port_id,omitempty"`       // 操作的港口ID（离港/到港），空则使用订单的出发港或目的港
	ActualTime string `json:"actual_time,omitempty"`   // 实际到港/离港时间，格式 "2006-01-02 15:04:05"
	Notes      string `json:"notes,omitempty"`         // 操作备注
}

// UpdateOrderStatus 更新订单状态。
//
// 使用状态机（biz.OrderStateMachine）校验状态转换合法性。
// 状态变更后会通过 WebSocket 向该订单货主推送通知。
// 仅 shipping 和 admin 角色可调用。shipping 角色只能更新自己公司的订单。
func (h *OrderHandler) UpdateOrderStatus(c *gin.Context) {
	role, _ := c.Get("role")
	if role != "shipping" && role != "admin" {
		response.ErrorWithCode(c.Writer, errors.CodeForbidden, "only shipping or admin can update order status")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid order id")
		return
	}

	if companyID, ok := getShippingCompanyID(c); ok {
		belongs, err := h.svc.CheckOrderBelongsToShippingCompany(c.Request.Context(), id, companyID)
		if err != nil || !belongs {
			response.ErrorWithCode(c.Writer, errors.CodeNotFound, "order not found")
			return
		}
	}
	var req updateOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c.Writer, "invalid request body")
		return
	}
	if err := validator.Validate(req); err != nil {
		response.BadRequest(c.Writer, err.Error())
		return
	}
	var actualTime *time.Time
	if req.ActualTime != "" {
		t, err := time.Parse("2006-01-02 15:04:05", req.ActualTime)
		if err == nil {
			actualTime = &t
		}
	}
	var portID *int64
	if req.PortID > 0 {
		portID = &req.PortID
	}
	err = h.svc.UpdateOrderStatus(c.Request.Context(), id, req.Status, actualTime, req.Notes, portID)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.ErrorWithCode(c.Writer, appErr.Code, appErr.Message)
			return
		}
		response.InternalServerError(c.Writer, "failed to update order status")
		return
	}
	response.Success(c.Writer, gin.H{"message": "order status updated"})
}

// RecordPortVisit 记录运输中订单在某个港口的到港/离港时间及货物操作。
func (h *OrderHandler) RecordPortVisit(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid order id")
		return
	}
	if companyID, ok := getShippingCompanyID(c); ok {
		belongs, err := h.svc.CheckOrderBelongsToShippingCompany(c.Request.Context(), id, companyID)
		if err != nil || !belongs {
			response.ErrorWithCode(c.Writer, errors.CodeNotFound, "order not found")
			return
		}
	}
	var req service.PortVisitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c.Writer, "invalid request body")
		return
	}
	if err := h.svc.RecordPortVisit(c.Request.Context(), id, &req); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.ErrorWithCode(c.Writer, appErr.Code, appErr.Message)
			return
		}
		response.InternalServerError(c.Writer, "failed to record port visit")
		return
	}
	response.Success(c.Writer, gin.H{"message": "port visit recorded"})
}

// ListOrders 分页查询订单列表。
//
// 权限校验：
//   - shipper 角色：只能查自己的订单（user_id == shipper_company_id）。
//   - shipping 角色：只能查自己公司的订单（通过 load_note → line → shipping_company_id 关联）。
//   - admin 角色：可查任意货主的订单。
//
// 查询参数：page, page_size, order_no（模糊搜索）, order_status（精确匹配）, sort_by, sort_order。
func (h *OrderHandler) ListOrders(c *gin.Context) {
	role, _ := c.Get("role")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	sortBy := c.DefaultQuery("sort_by", "create_time")
	sortOrder := c.DefaultQuery("sort_order", "desc")

	req := service.PageRequest{
		Page:        page,
		PageSize:    pageSize,
		SortBy:      sortBy,
		SortOrder:   sortOrder,
		AllowedSort: service.DefaultOrderSortFields(),
	}

	// 搜索过滤条件
	orderNo := c.Query("order_no")
	orderStatusStr := c.Query("order_status")
	var orderStatus *int8
	if orderStatusStr != "" {
		if s, err := strconv.Atoi(orderStatusStr); err == nil {
			v := int8(s)
			orderStatus = &v
		}
	}

	switch role {
	case "shipper":
		uid, ok := getUserID(c)
		if !ok {
			response.ErrorWithCode(c.Writer, errors.CodeForbidden, "invalid user identity")
			return
		}
		orders, total, err := h.svc.ListOrdersByShipper(c.Request.Context(), uid, req, orderNo, orderStatus)
		if err != nil {
			response.InternalServerError(c.Writer, "failed to list orders")
			return
		}
		response.SuccessPage(c.Writer, orders, page, pageSize, total)

	case "shipping":
		uid, ok := getUserID(c)
		if !ok {
			response.ErrorWithCode(c.Writer, errors.CodeForbidden, "invalid user identity")
			return
		}
		orders, total, err := h.svc.ListOrdersByShippingCompany(c.Request.Context(), uid, req, orderNo, orderStatus)
		if err != nil {
			response.InternalServerError(c.Writer, "failed to list orders")
			return
		}
		response.SuccessPage(c.Writer, orders, page, pageSize, total)

	default: // admin
		orders, total, err := h.svc.ListAllOrders(c.Request.Context(), req, orderNo, orderStatus)
		if err != nil {
			response.InternalServerError(c.Writer, "failed to list orders")
			return
		}
		response.SuccessPage(c.Writer, orders, page, pageSize, total)
	}
}

// GetOrderTracking 查询订单的实时追踪信息。
//
// 返回包含：订单状态、装卸货时间、起止港名称、靠泊计划/实际时间、
// 船舶名称、航线名称等物流追踪信息。
// 权限：shipping 角色只能查自己公司的订单追踪。
func (h *OrderHandler) GetOrderTracking(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid order id")
		return
	}

	if companyID, ok := getShippingCompanyID(c); ok {
		belongs, err := h.svc.CheckOrderBelongsToShippingCompany(c.Request.Context(), id, companyID)
		if err != nil || !belongs {
			response.ErrorWithCode(c.Writer, errors.CodeNotFound, "order not found")
			return
		}
	}

	if role, _ := c.Get("role"); role == "shipper" {
		if uid, ok := c.Get("user_id"); ok {
			if shipperID, ok2 := uid.(int64); ok2 {
				belongs, err := h.svc.CheckOrderBelongsToShipper(c.Request.Context(), id, shipperID)
				if err != nil || !belongs {
					response.ErrorWithCode(c.Writer, errors.CodeNotFound, "order not found")
					return
				}
			}
		}
	}

	tracking, err := h.svc.GetOrderTracking(c.Request.Context(), id)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.ErrorWithCode(c.Writer, appErr.Code, appErr.Message)
			return
		}
		response.InternalServerError(c.Writer, err.Error())
		return
	}
	response.Success(c.Writer, tracking)
}

// PayOrder 虚拟支付——更新订单支付状态为已支付。
func (h *OrderHandler) PayOrder(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil { response.BadRequest(c.Writer, "invalid order id"); return }
	role, _ := c.Get("role")
	if role != "shipper" && role != "admin" {
		response.ErrorWithCode(c.Writer, errors.CodeForbidden, "only shipper can pay"); return
	}
	if role == "shipper" {
		if uid, ok := c.Get("user_id"); ok {
			if sid, ok2 := uid.(int64); ok2 {
				belongs, err := h.svc.CheckOrderBelongsToShipper(c.Request.Context(), id, sid)
				if err != nil || !belongs {
					response.ErrorWithCode(c.Writer, errors.CodeNotFound, "order not found"); return
				}
			}
		}
	}
	err = h.svc.PayOrder(c.Request.Context(), id)
	if err != nil { response.InternalServerError(c.Writer, "failed to pay order"); return }
	response.Success(c.Writer, gin.H{"message": "payment successful"})
}
