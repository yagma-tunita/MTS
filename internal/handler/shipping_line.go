package handler

import (
	"strconv"

	"backend/internal/model"
	"backend/internal/service"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
)

type ShippingLineHandler struct {
	svc service.ShippingLineService
}

func NewShippingLineHandler(svc service.ShippingLineService) *ShippingLineHandler {
	return &ShippingLineHandler{svc: svc}
}

func (h *ShippingLineHandler) GetLine(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid line id")
		return
	}
	line, err := h.svc.GetLineByID(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c.Writer, "shipping line not found")
		return
	}
	response.Success(c.Writer, line)
}

func (h *ShippingLineHandler) ListLines(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 { pageSize = 20 }; if pageSize > 100 { pageSize = 100 }

	// shipping 角色只能看自己的航线
	role, _ := c.Get("role")
	if role == "shipping" {
		if uid, ok := c.Get("user_id"); ok {
			if companyID, ok2 := uid.(int64); ok2 {
				lines, total, err := h.svc.ListLinesByCompany(c.Request.Context(), companyID, page, pageSize)
				if err != nil {
					response.InternalServerError(c.Writer, "failed to list lines")
					return
				}
				response.SuccessPage(c.Writer, lines, page, pageSize, total)
				return
			}
		}
	}

	keyword := c.Query("keyword")
	lines, total, err := h.svc.ListLines(c.Request.Context(), page, pageSize, keyword)
	if err != nil {
		response.InternalServerError(c.Writer, "failed to list lines")
		return
	}
	response.SuccessPage(c.Writer, lines, page, pageSize, total)
}

func (h *ShippingLineHandler) GetPortSequence(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid line id")
		return
	}
	portIDs, err := h.svc.GetPortSequence(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c.Writer, "port sequence not found")
		return
	}
	response.Success(c.Writer, gin.H{"port_sequence": portIDs})
}

func (h *ShippingLineHandler) CreateLine(c *gin.Context) {
	var line model.ShippingLine
	if err := c.ShouldBindJSON(&line); err != nil {
		response.BadRequest(c.Writer, "invalid request body")
		return
	}
	if err := h.svc.CreateLine(c.Request.Context(), &line); err != nil {
		response.InternalServerError(c.Writer, "failed to create line")
		return
	}
	response.Success(c.Writer, line)
}

func (h *ShippingLineHandler) UpdateLine(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid line id")
		return
	}
	existing, err := h.svc.GetLineByID(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c.Writer, "line not found")
		return
	}
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c.Writer, "invalid request body")
		return
	}
	if v, ok := req["line_name"]; ok && v != nil { existing.LineName = v.(string) }
	if v, ok := req["port_sequence"]; ok && v != nil { s := v.(string); existing.PortSequence = &s }
	if v, ok := req["total_distance_nm"]; ok && v != nil { f := v.(float64); existing.TotalDistanceNm = &f }
	if v, ok := req["departure_port_name"]; ok && v != nil { existing.DeparturePortName = strPtr(v.(string)) }
	if v, ok := req["destination_port_name"]; ok && v != nil { existing.DestinationPortName = strPtr(v.(string)) }
	if v, ok := req["description"]; ok && v != nil { existing.Description = strPtr(v.(string)) }
	if err := h.svc.UpdateLine(c.Request.Context(), existing); err != nil {
		response.InternalServerError(c.Writer, "failed to update line")
		return
	}
	response.Success(c.Writer, existing)
}

func (h *ShippingLineHandler) DeleteLine(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid line id")
		return
	}
	if err := h.svc.DeleteLine(c.Request.Context(), id); err != nil {
		response.InternalServerError(c.Writer, "failed to delete line")
		return
	}
	response.Success(c.Writer, gin.H{"message": "line deleted"})
}
