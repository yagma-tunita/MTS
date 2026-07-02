package handler

import (
	"strconv"

	"backend/internal/service"
	"backend/pkg/errors"
	"backend/pkg/response"
	"backend/pkg/validator"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	svc service.OrderService
}

func NewOrderHandler(svc service.OrderService) *OrderHandler {
	return &OrderHandler{svc: svc}
}

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

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c.Writer, "invalid request body")
		return
	}
	if err := validator.Validate(req); err != nil {
		response.BadRequest(c.Writer, err.Error())
		return
	}

	role, _ := c.Get("role")
	userID, _ := c.Get("user_id")
	if role == "shipper" {
		if uid, ok := userID.(int64); !ok || uid != req.ShipperCompanyID {
			response.ErrorWithCode(c.Writer, errors.CodeForbidden, "shipper_company_id mismatch")
			return
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

func (h *OrderHandler) GetOrder(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid order id")
		return
	}
	order, err := h.svc.GetOrderByID(c.Request.Context(), id)
	if err != nil {
		response.ErrorWithCode(c.Writer, errors.CodeNotFound, "order not found")
		return
	}
	response.Success(c.Writer, order)
}

func (h *OrderHandler) CancelOrder(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid order id")
		return
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

type updateOrderStatusRequest struct {
	Status int8 `json:"status" validate:"required,min=0,max=4"`
}

func (h *OrderHandler) UpdateOrderStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid order id")
		return
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
	err = h.svc.UpdateOrderStatus(c.Request.Context(), id, req.Status)
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

func (h *OrderHandler) ListOrders(c *gin.Context) {
	shipperIDStr := c.Query("shipper_company_id")
	if shipperIDStr == "" {
		response.BadRequest(c.Writer, "shipper_company_id is required")
		return
	}
	shipperID, err := strconv.ParseInt(shipperIDStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid shipper_company_id")
		return
	}
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

	orders, total, err := h.svc.ListOrdersByShipper(c.Request.Context(), shipperID, req)
	if err != nil {
		response.InternalServerError(c.Writer, "failed to list orders")
		return
	}
	response.SuccessPage(c.Writer, orders, page, pageSize, total)
}

func (h *OrderHandler) GetOrderTracking(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid order id")
		return
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
