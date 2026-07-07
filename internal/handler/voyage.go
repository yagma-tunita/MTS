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
