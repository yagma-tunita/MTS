package handler

import (
	"strconv"

	"backend/internal/model"
	"backend/internal/service"
	"backend/pkg/errors"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ShippingLineHandler struct {
	svc service.ShippingLineService
	db  *gorm.DB
}

func NewShippingLineHandler(svc service.ShippingLineService, db *gorm.DB) *ShippingLineHandler {
	return &ShippingLineHandler{svc: svc, db: db}
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

	role, _ := c.Get("role")
	keyword := c.Query("keyword")

	// shipping 角色：看自己所有状态的航线
	if role == "shipping" {
		if uid, ok := c.Get("user_id"); ok {
			if companyID, ok2 := uid.(int64); ok2 {
				lines, total, err := h.svc.ListLinesByCompany(c.Request.Context(), companyID, page, pageSize, nil)
				if err != nil {
					response.InternalServerError(c.Writer, "failed to list lines")
					return
				}
				response.SuccessPage(c.Writer, lines, page, pageSize, total)
				return
			}
		}
	}

	// 货主/shipper角色：只看已启用的航线
	active := int8(model.LineStatusActive)
	lines, total, err := h.svc.ListLines(c.Request.Context(), page, pageSize, keyword, &active)
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
	role, _ := c.Get("role")
	if role == "shipping" {
		v := int8(model.LineStatusPending)
		line.LineStatus = &v
		if uid, ok := c.Get("user_id"); ok {
			if companyID, ok2 := uid.(int64); ok2 {
				line.ShippingCompanyID = &companyID
			}
		}
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
	role, _ := c.Get("role")
	if role == "shipping" {
		uid, ok := c.Get("user_id")
		if !ok {
			response.ErrorWithCode(c.Writer, errors.CodeForbidden, "invalid user identity")
			return
		}
		companyID, ok := uid.(int64)
		if !ok {
			response.ErrorWithCode(c.Writer, errors.CodeForbidden, "invalid user identity")
			return
		}
		line, err := h.svc.GetLineByID(c.Request.Context(), id)
		if err != nil || line.ShippingCompanyID == nil || *line.ShippingCompanyID != companyID {
			response.NotFound(c.Writer, "line not found")
			return
		}
		if err := h.db.Model(&model.ShippingLine{}).Where("line_id = ?", id).Update("line_status", model.LineStatusDeprecated).Error; err != nil {
			response.InternalServerError(c.Writer, "failed to deprecate line")
			return
		}
		response.Success(c.Writer, gin.H{"message": "line deprecated"})
		return
	}
	if err := h.svc.DeleteLine(c.Request.Context(), id); err != nil {
		response.InternalServerError(c.Writer, "failed to delete line")
		return
	}
	response.Success(c.Writer, gin.H{"message": "line deleted"})
}

func (h *ShippingLineHandler) ListPendingLines(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 { pageSize = 20 }; if pageSize > 100 { pageSize = 100 }

	pending := int8(model.LineStatusPending)
	lines, total, err := h.svc.ListLines(c.Request.Context(), page, pageSize, "", &pending)
	if err != nil {
		response.InternalServerError(c.Writer, "failed to list pending lines")
		return
	}
	response.SuccessPage(c.Writer, lines, page, pageSize, total)
}

func (h *ShippingLineHandler) ApproveLine(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid line id")
		return
	}
	if err := h.db.Model(&model.ShippingLine{}).Where("line_id = ?", id).Update("line_status", model.LineStatusActive).Error; err != nil {
		response.InternalServerError(c.Writer, "failed to approve line")
		return
	}
	response.Success(c.Writer, gin.H{"message": "line approved"})
}

func (h *ShippingLineHandler) DeprecateLine(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid line id")
		return
	}
	if err := h.db.Model(&model.ShippingLine{}).Where("line_id = ?", id).Update("line_status", model.LineStatusDeprecated).Error; err != nil {
		response.InternalServerError(c.Writer, "failed to deprecate line")
		return
	}
	response.Success(c.Writer, gin.H{"message": "line deprecated"})
}

func (h *ShippingLineHandler) GetAssignedVessels(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid line id")
		return
	}
	var vesselIDs []int64
	if err := h.db.Model(&model.LineVessel{}).Where("line_id = ?", id).Pluck("vessel_id", &vesselIDs).Error; err != nil {
		response.InternalServerError(c.Writer, "failed to get assigned vessels")
		return
	}
	response.Success(c.Writer, gin.H{"vessel_ids": vesselIDs})
}

func (h *ShippingLineHandler) AssignVessel(c *gin.Context) {
	lineIDStr := c.Param("id")
	lineID, err := strconv.ParseInt(lineIDStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid line id")
		return
	}
	var req struct {
		VesselID int64 `json:"vessel_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c.Writer, "invalid request body")
		return
	}
	if err := h.db.Create(&model.LineVessel{LineID: lineID, VesselID: req.VesselID}).Error; err != nil {
		response.InternalServerError(c.Writer, "failed to assign vessel")
		return
	}
	response.Success(c.Writer, gin.H{"message": "vessel assigned"})
}

func (h *ShippingLineHandler) UnassignVessel(c *gin.Context) {
	lineIDStr := c.Param("id")
	lineID, err := strconv.ParseInt(lineIDStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid line id")
		return
	}
	vesselIDStr := c.Param("vesselId")
	vesselID, err := strconv.ParseInt(vesselIDStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid vessel id")
		return
	}
	if err := h.db.Delete(&model.LineVessel{}, "line_id = ? AND vessel_id = ?", lineID, vesselID).Error; err != nil {
		response.InternalServerError(c.Writer, "failed to unassign vessel")
		return
	}
	response.Success(c.Writer, gin.H{"message": "vessel unassigned"})
}

func (h *ShippingLineHandler) ReactivateLine(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid line id")
		return
	}
	uid, ok := c.Get("user_id")
	if !ok {
		response.ErrorWithCode(c.Writer, errors.CodeForbidden, "invalid user identity")
		return
	}
	companyID, ok := uid.(int64)
	if !ok {
		response.ErrorWithCode(c.Writer, errors.CodeForbidden, "invalid user identity")
		return
	}
	line, err := h.svc.GetLineByID(c.Request.Context(), id)
	if err != nil || line.ShippingCompanyID == nil || *line.ShippingCompanyID != companyID {
		response.NotFound(c.Writer, "line not found")
		return
	}
	if line.LineStatus == nil || *line.LineStatus != model.LineStatusDeprecated {
		response.BadRequest(c.Writer, "only deprecated lines can be reactivated")
		return
	}
	if err := h.db.Model(&model.ShippingLine{}).Where("line_id = ?", id).Update("line_status", model.LineStatusPending).Error; err != nil {
		response.InternalServerError(c.Writer, "failed to reactivate line")
		return
	}
	response.Success(c.Writer, gin.H{"message": "line reactivated"})
}
