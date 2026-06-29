package handler

import (
	"strconv"
	"time"

	"backend/internal/service"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
)

type ReportHandler struct {
	svc service.ReportService
}

func NewReportHandler(svc service.ReportService) *ReportHandler {
	return &ReportHandler{svc: svc}
}

func (h *ReportHandler) OrderStatistics(c *gin.Context) {
	startStr := c.Query("start_date")
	endStr := c.Query("end_date")
	if startStr == "" || endStr == "" {
		response.BadRequest(c.Writer, "start_date and end_date are required")
		return
	}
	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		response.BadRequest(c.Writer, "invalid start_date")
		return
	}
	end, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		response.BadRequest(c.Writer, "invalid end_date")
		return
	}
	end = end.Add(24*time.Hour - time.Second)

	stats, err := h.svc.OrderStatistics(c.Request.Context(), start, end)
	if err != nil {
		response.InternalServerError(c.Writer, err.Error())
		return
	}
	response.Success(c.Writer, stats)
}

func (h *ReportHandler) VoyageUtilization(c *gin.Context) {
	lineIDStr := c.Query("line_id")
	vesselIDStr := c.Query("vessel_id")
	dateStr := c.Query("voyage_date")
	if lineIDStr == "" || vesselIDStr == "" || dateStr == "" {
		response.BadRequest(c.Writer, "line_id, vessel_id, voyage_date are required")
		return
	}
	lineID, err := strconv.ParseInt(lineIDStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid line_id")
		return
	}
	vesselID, err := strconv.ParseInt(vesselIDStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid vessel_id")
		return
	}
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		response.BadRequest(c.Writer, "invalid voyage_date")
		return
	}

	util, err := h.svc.VoyageUtilization(c.Request.Context(), lineID, vesselID, date)
	if err != nil {
		response.InternalServerError(c.Writer, err.Error())
		return
	}
	response.Success(c.Writer, util)
}
