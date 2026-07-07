package handler

import (
	"strconv"

	"backend/internal/model"
	"backend/internal/service"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
)

func strPtr(s string) *string { return &s }

// PortHandler 处理港口相关请求（查询单个、列表、按城市查询）。
//
// 数据来自 port_service，底层使用缓存（10 分钟 TTL）。
type PortHandler struct {
	svc service.PortService
}

// NewPortHandler 创建港口查询处理器
func NewPortHandler(svc service.PortService) *PortHandler {
	return &PortHandler{svc: svc}
}

// GetPort 根据港口 ID 查询详情。
// 返回数据包含城市信息（city_id + city_name）通过 GORM Preload 加载。
func (h *PortHandler) GetPort(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid port id")
		return
	}
	port, err := h.svc.GetPortByID(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c.Writer, "port not found")
		return
	}
	response.Success(c.Writer, port)
}

// ListPorts 分页查询港口列表，可选按 city_id 筛选。
// 使用缓存，key 为 "ports:list:{page}:{pageSize}"，TTL 10 分钟。
func (h *PortHandler) ListPorts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 { pageSize = 20 }; if pageSize > 100 { pageSize = 100 }

	cityIDStr := c.Query("city_id")
	if cityIDStr != "" {
		cityID, err := strconv.ParseInt(cityIDStr, 10, 64)
		if err == nil {
			ports, total, err := h.svc.ListPortsByCity(c.Request.Context(), cityID, page, pageSize)
			if err != nil {
				response.InternalServerError(c.Writer, "failed to list ports")
				return
			}
			response.SuccessPage(c.Writer, ports, page, pageSize, total)
			return
		}
	}

	keyword := c.Query("keyword")
	ports, total, err := h.svc.ListPorts(c.Request.Context(), page, pageSize, keyword)
	if err != nil {
		response.InternalServerError(c.Writer, "failed to list ports")
		return
	}
	response.SuccessPage(c.Writer, ports, page, pageSize, total)
}

// CreatePort 管理员创建港口。
func (h *PortHandler) CreatePort(c *gin.Context) {
	var port model.Port
	if err := c.ShouldBindJSON(&port); err != nil {
		response.BadRequest(c.Writer, "invalid request body")
		return
	}
	if err := h.svc.CreatePort(c.Request.Context(), &port); err != nil {
		response.InternalServerError(c.Writer, "failed to create port")
		return
	}
	response.Success(c.Writer, port)
}

// UpdatePort 管理员更新港口。
func (h *PortHandler) UpdatePort(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid port id")
		return
	}
	existing, err := h.svc.GetPortByID(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c.Writer, "port not found")
		return
	}
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c.Writer, "invalid request body")
		return
	}
	if v, ok := req["port_name"]; ok { existing.PortName = v.(string) }
	if v, ok := req["port_code"]; ok { existing.PortCode = strPtr(v.(string)) }
	if v, ok := req["city_id"]; ok && v != nil { idVal := int64(v.(float64)); existing.CityID = &idVal }
	if v, ok := req["latitude"]; ok && v != nil { f := v.(float64); existing.Latitude = &f }
	if v, ok := req["longitude"]; ok && v != nil { f := v.(float64); existing.Longitude = &f }
	if v, ok := req["port_type"]; ok && v != nil { existing.PortType = strPtr(v.(string)) }
	if v, ok := req["max_draft_meter"]; ok && v != nil { f := v.(float64); existing.MaxDraftMeter = &f }
	if err := h.svc.UpdatePort(c.Request.Context(), existing); err != nil {
		response.InternalServerError(c.Writer, "failed to update port")
		return
	}
	response.Success(c.Writer, existing)
}

// DeletePort 管理员删除港口。
func (h *PortHandler) DeletePort(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid port id")
		return
	}
	if err := h.svc.DeletePort(c.Request.Context(), id); err != nil {
		response.InternalServerError(c.Writer, "failed to delete port")
		return
	}
	response.Success(c.Writer, gin.H{"message": "port deleted"})
}

// ListPortsByCity 根据城市 ID 分页查询港口。
// 使用缓存，key 为 "ports:city:{cityID}:{page}:{pageSize}"，TTL 10 分钟。
func (h *PortHandler) ListPortsByCity(c *gin.Context) {
	cityIDStr := c.Query("city_id")
	if cityIDStr == "" {
		response.BadRequest(c.Writer, "city_id is required")
		return
	}
	cityID, err := strconv.ParseInt(cityIDStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid city_id")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 { pageSize = 20 }; if pageSize > 100 { pageSize = 100 }
	ports, total, err := h.svc.ListPortsByCity(c.Request.Context(), cityID, page, pageSize)
	if err != nil {
		response.InternalServerError(c.Writer, "failed to list ports")
		return
	}
	response.SuccessPage(c.Writer, ports, page, pageSize, total)
}
