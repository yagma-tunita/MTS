package handler

import (
	"strconv"
	"time"

	"backend/internal/service"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
)

// ReportHandler 处理报表统计请求。
type ReportHandler struct {
	svc service.ReportService
}

// NewReportHandler 创建报表统计处理器
func NewReportHandler(svc service.ReportService) *ReportHandler {
	return &ReportHandler{svc: svc}
}

// OrderStatistics 获取指定时间范围内的订单统计数据。
//
// 参数：
//   - start_date: 开始日期（含），格式 yyyy-MM-dd。
//   - end_date: 结束日期（含），格式 yyyy-MM-dd。
//
// 返回：总订单数、总重量、总体积、总运费、已完成/已取消/运输中数量。
//
// 权限：shipping 角色只能看自己公司的统计。
// 注意：end_date 会被自动扩展为当天的 23:59:59，确保包含全天。
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

	var shippingCompanyID int64
	if cid, ok := getShippingCompanyID(c); ok {
		shippingCompanyID = cid
	}
	var shipperCompanyID int64
	if role, _ := c.Get("role"); role == "shipper" {
		if uid, ok := c.Get("user_id"); ok {
			if id, ok2 := uid.(int64); ok2 {
				shipperCompanyID = id
			}
		}
	}

	stats, err := h.svc.OrderStatistics(c.Request.Context(), start, end, shippingCompanyID, shipperCompanyID)
	if err != nil {
		response.InternalServerError(c.Writer, err.Error())
		return
	}
	response.Success(c.Writer, stats)
}

// VoyageUtilization 获取指定航次的舱位利用率。
//
// 参数：
//   - line_id: 航线 ID。
//   - vessel_id: 船舶 ID。
//   - voyage_date: 航次日期，格式 yyyy-MM-dd。
//
// 返回：最大载重、已占用吨位、利用率百分比。
// 利用率 = occupied / max_capacity × 100。
// 权限：shipping 角色只能查自己公司船舶的利用率。
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

	var shippingCompanyID int64
	if cid, ok := getShippingCompanyID(c); ok {
		shippingCompanyID = cid
	}

	util, err := h.svc.VoyageUtilization(c.Request.Context(), lineID, vesselID, date, shippingCompanyID)
	if err != nil {
		response.InternalServerError(c.Writer, err.Error())
		return
	}
	response.Success(c.Writer, util)
}
