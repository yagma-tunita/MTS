package handler

import (
	"strconv"
	"time"

	"backend/internal/model"
	"backend/internal/service"
	"backend/pkg/errors"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
)

type VoyageHandler struct {
	svc service.VoyageService
}

func NewVoyageHandler(svc service.VoyageService) *VoyageHandler {
	return &VoyageHandler{svc: svc}
}

// createVoyageRequest 批量创建航次请求
type createVoyageRequest struct {
	LineID     int64                   `json:"line_id" binding:"required"`
	VesselID   int64                   `json:"vessel_id" binding:"required"`
	VoyageDate string                  `json:"voyage_date" binding:"required"`
	UnitPrice  *float64                `json:"unit_price"`
	PortStops  []createVoyagePortStop  `json:"port_stops" binding:"required,min=2"`
}

type createVoyagePortStop struct {
	PortID             int64      `json:"port_id" binding:"required"`
	PlannedArrivalTime   *time.Time `json:"planned_arrival_time"`
	PlannedDepartureTime *time.Time `json:"planned_departure_time"`
}

func (h *VoyageHandler) CreateVoyage(c *gin.Context) {
	role, _ := c.Get("role")
	if role != "shipping" && role != "admin" {
		response.ErrorWithCode(c.Writer, errors.CodeForbidden, "only shipping companies can create voyages"); return
	}
	var req createVoyageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c.Writer, "invalid request body: "+err.Error())
		return
	}
	voyageDate, err := time.Parse("2006-01-02", req.VoyageDate)
	if err != nil {
		response.BadRequest(c.Writer, "invalid voyage_date, use yyyy-MM-dd")
		return
	}
	berthings := make([]model.VoyageBerthing, len(req.PortStops))
	for i, ps := range req.PortStops {
		berthings[i] = model.VoyageBerthing{
			LineID:               &req.LineID,
			VesselID:             &req.VesselID,
			VoyageDate:           voyageDate,
			SequenceNo:           int32(i + 1),
			PortID:               &ps.PortID,
			PlannedArrivalTime:   ps.PlannedArrivalTime,
			PlannedDepartureTime: ps.PlannedDepartureTime,
		}
	}
	if err := h.svc.CreateVoyage(c.Request.Context(), req.LineID, req.VesselID, voyageDate, berthings, req.UnitPrice); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.ErrorWithCode(c.Writer, appErr.Code, appErr.Message)
			return
		}
		response.InternalServerError(c.Writer, "failed to create voyage")
		return
	}
	response.Success(c.Writer, gin.H{"message": "voyage created", "port_count": len(berthings)})
}

func (h *VoyageHandler) ListVoyages(c *gin.Context) {
	role, _ := c.Get("role")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 { pageSize = 20 }; if pageSize > 100 { pageSize = 100 }

	var voyages []service.VoyageGroup
	var total int64
	var err error

	if role == "shipping" {
		if uid, ok := c.Get("user_id"); ok {
			if companyID, ok2 := uid.(int64); ok2 {
				voyages, total, err = h.svc.ListVoyagesByCompany(c.Request.Context(), companyID, page, pageSize)
				if err != nil {
					response.InternalServerError(c.Writer, "failed to list voyages")
					return
				}
				response.SuccessPage(c.Writer, voyages, page, pageSize, total)
				return
			}
		}
	}
	voyages, total, err = h.svc.ListVoyages(c.Request.Context(), page, pageSize)
	if err != nil {
		response.InternalServerError(c.Writer, "failed to list voyages")
		return
	}
	response.SuccessPage(c.Writer, voyages, page, pageSize, total)
}

func (h *VoyageHandler) GetVoyageDetail(c *gin.Context) {
	lineIDStr := c.Query("line_id")
	vesselIDStr := c.Query("vessel_id")
	voyageDateStr := c.Query("voyage_date")
	if lineIDStr == "" || vesselIDStr == "" || voyageDateStr == "" {
		response.BadRequest(c.Writer, "missing required params: line_id, vessel_id, voyage_date")
		return
	}
	lineID, _ := strconv.ParseInt(lineIDStr, 10, 64)
	vesselID, _ := strconv.ParseInt(vesselIDStr, 10, 64)
	voyageDate, err := time.Parse("2006-01-02", voyageDateStr)
	if err != nil {
		response.BadRequest(c.Writer, "invalid voyage_date")
		return
	}
	detail, err := h.svc.GetVoyageDetail(c.Request.Context(), lineID, vesselID, voyageDate)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.ErrorWithCode(c.Writer, appErr.Code, appErr.Message)
			return
		}
		response.InternalServerError(c.Writer, "failed to get voyage detail")
		return
	}
	response.Success(c.Writer, detail)
}

func (h *VoyageHandler) Recommend(c *gin.Context) {
	startPortStr := c.Query("start_port_id")
	endPortStr := c.Query("end_port_id")
	requiredTonStr := c.Query("required_ton")

	if startPortStr == "" || endPortStr == "" || requiredTonStr == "" {
		response.BadRequest(c.Writer, "missing required parameters: start_port_id, end_port_id, required_ton")
		return
	}

	startPortID, err := strconv.ParseInt(startPortStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid start_port_id")
		return
	}
	endPortID, err := strconv.ParseInt(endPortStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid end_port_id")
		return
	}
	requiredTon, err := strconv.ParseFloat(requiredTonStr, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid required_ton")
		return
	}

	recommendations, err := h.svc.RecommendVoyages(c.Request.Context(), startPortID, endPortID, requiredTon)
	if err != nil {
		response.InternalServerError(c.Writer, "failed to recommend voyages")
		return
	}
	response.Success(c.Writer, recommendations)
}

// createVoyageBerthingRequest 创建航次靠泊记录请求。
type createVoyageBerthingRequest struct {
	LineID             *int64     `json:"line_id" validate:"required"`
	VesselID           *int64     `json:"vessel_id" validate:"required"`
	VoyageDate         string     `json:"voyage_date" validate:"required"`
	SequenceNo         int32      `json:"sequence_no"`
	PortID             *int64     `json:"port_id" validate:"required"`
	BerthID            *int64     `json:"berth_id"`
	PlannedArrivalTime   *time.Time `json:"planned_arrival_time"`
	PlannedDepartureTime *time.Time `json:"planned_departure_time"`
	UnitPrice          *float64   `json:"unit_price"` // 单价(元/吨)，用于运费计算
}

func (h *VoyageHandler) CreateVoyageBerthing(c *gin.Context) {
	role, _ := c.Get("role")
	if role != "shipping" && role != "admin" {
		response.ErrorWithCode(c.Writer, errors.CodeForbidden, "only shipping companies can create voyages"); return
	}
	var req createVoyageBerthingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c.Writer, "invalid request body")
		return
	}
	voyageDate, err := time.Parse("2006-01-02", req.VoyageDate)
	if err != nil {
		response.BadRequest(c.Writer, "invalid voyage_date, use yyyy-MM-dd")
		return
	}
	berthing := &model.VoyageBerthing{
		LineID:               req.LineID,
		VesselID:             req.VesselID,
		VoyageDate:           voyageDate,
		SequenceNo:           req.SequenceNo,
		PortID:               req.PortID,
		BerthID:              req.BerthID,
		PlannedArrivalTime:   req.PlannedArrivalTime,
		PlannedDepartureTime: req.PlannedDepartureTime,
	}
	if err := h.svc.CreateVoyageBerthing(c.Request.Context(), berthing, req.UnitPrice); err != nil {
		response.InternalServerError(c.Writer, "failed to create voyage berthing")
		return
	}
	response.Success(c.Writer, berthing)
}
