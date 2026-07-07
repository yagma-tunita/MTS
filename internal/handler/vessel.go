package handler

import (
	"strconv"

	"backend/internal/model"
	"backend/internal/service"
	"backend/pkg/errors"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
)

type VesselHandler struct {
	svc service.VesselService
}

func NewVesselHandler(svc service.VesselService) *VesselHandler {
	return &VesselHandler{svc: svc}
}

func (h *VesselHandler) GetVessel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid vessel id")
		return
	}
	vessel, err := h.svc.GetVesselByID(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c.Writer, "vessel not found")
		return
	}
	role, _ := c.Get("role")
	if role == "shipping" {
		if uid, ok := c.Get("user_id"); ok {
			if companyID, ok2 := uid.(int64); ok2 {
				if vessel.ShippingCompanyID == nil || *vessel.ShippingCompanyID != companyID {
					response.NotFound(c.Writer, "vessel not found"); return
				}
			}
		}
	}
	response.Success(c.Writer, vessel)
}

func (h *VesselHandler) ListVessels(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 { pageSize = 20 }; if pageSize > 100 { pageSize = 100 }

	// shipping 角色只能看自己的船
	role, _ := c.Get("role")
	if role == "shipping" {
		uid, ok := c.Get("user_id")
		if !ok {
			response.ErrorWithCode(c.Writer, errors.CodeForbidden, "invalid user identity"); return
		}
		companyID, ok := uid.(int64)
		if !ok {
			response.ErrorWithCode(c.Writer, errors.CodeForbidden, "invalid user identity"); return
		}
		vessels, total, err := h.svc.ListVesselsByCompany(c.Request.Context(), companyID, page, pageSize)
		if err != nil {
			response.InternalServerError(c.Writer, "failed to list vessels")
			return
		}
		response.SuccessPage(c.Writer, vessels, page, pageSize, total)
		return
	}

	companyIDStr := c.Query("shipping_company_id")
	if companyIDStr != "" {
		companyID, err := strconv.ParseInt(companyIDStr, 10, 64)
		if err == nil {
			vessels, total, err := h.svc.ListVesselsByCompany(c.Request.Context(), companyID, page, pageSize)
			if err != nil {
				response.InternalServerError(c.Writer, "failed to list vessels")
				return
			}
			response.SuccessPage(c.Writer, vessels, page, pageSize, total)
			return
		}
	}

	keyword := c.Query("keyword")
	vessels, total, err := h.svc.ListVessels(c.Request.Context(), page, pageSize, keyword)
	if err != nil {
		response.InternalServerError(c.Writer, "failed to list vessels")
		return
	}
	response.SuccessPage(c.Writer, vessels, page, pageSize, total)
}

func (h *VesselHandler) CreateVessel(c *gin.Context) {
	var vessel model.Vessel
	if err := c.ShouldBindJSON(&vessel); err != nil {
		response.BadRequest(c.Writer, "invalid request body")
		return
	}
	if err := h.svc.CreateVessel(c.Request.Context(), &vessel); err != nil {
		response.InternalServerError(c.Writer, "failed to create vessel")
		return
	}
	response.Success(c.Writer, vessel)
}

func (h *VesselHandler) UpdateVessel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid vessel id")
		return
	}
	existing, err := h.svc.GetVesselByID(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c.Writer, "vessel not found")
		return
	}
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c.Writer, "invalid request body")
		return
	}
	if v, ok := req["vessel_name"]; ok && v != nil { existing.VesselName = v.(string) }
	if v, ok := req["vessel_type"]; ok && v != nil { existing.VesselType = strPtr(v.(string)) }
	if v, ok := req["imo_number"]; ok && v != nil { existing.IMONumber = v.(string) }
	if v, ok := req["max_deadweight_ton"]; ok && v != nil { f := v.(float64); existing.MaxDeadweightTon = &f }
	if v, ok := req["gross_tonnage"]; ok && v != nil { f := v.(float64); existing.GrossTonnage = &f }
	if v, ok := req["speed_knot"]; ok && v != nil { f := v.(float64); existing.SpeedKnot = &f }
	if v, ok := req["container_teu"]; ok && v != nil { i := int32(v.(float64)); existing.ContainerTEU = &i }
	if v, ok := req["is_available"]; ok && v != nil { existing.IsAvailable = int8(v.(float64)) }
	if err := h.svc.UpdateVessel(c.Request.Context(), existing); err != nil {
		response.InternalServerError(c.Writer, "failed to update vessel")
		return
	}
	response.Success(c.Writer, existing)
}

func (h *VesselHandler) DeleteVessel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid vessel id")
		return
	}
	if err := h.svc.DeleteVessel(c.Request.Context(), id); err != nil {
		response.InternalServerError(c.Writer, "failed to delete vessel")
		return
	}
	response.Success(c.Writer, gin.H{"message": "vessel deleted"})
}
