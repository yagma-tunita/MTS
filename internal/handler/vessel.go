package handler

import (
	"strconv"

	"backend/internal/service"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
)

type VesselHandler struct {
	svc service.VesselService
}

func NewVesselHandler(svc service.VesselService) *VesselHandler {
	return &VesselHandler{svc: svc}
}

// GetVessel retrieves a vessel by its ID.
// @Summary      Get vessel by ID
// @Description  Return details of a specific vessel
// @Tags         Vessel
// @Produce      json
// @Param        id path int true "Vessel ID"
// @Success      200 {object} response.Response{data=model.Vessel} "Vessel found"
// @Failure      400 {object} response.Response "Invalid ID"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      404 {object} response.Response "Vessel not found"
// @Security     BearerAuth
// @Router       /vessels/{id} [get]
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
	response.Success(c.Writer, vessel)
}

// ListVessels returns a paginated list of vessels.
// @Summary      List vessels
// @Description  Get all vessels with pagination
// @Tags         Vessel
// @Produce      json
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(20)
// @Success      200 {object} response.PageResponse{data=[]model.Vessel} "Vessels list"
// @Failure      401 {object} response.Response "Unauthorized"
// @Security     BearerAuth
// @Router       /vessels [get]
func (h *VesselHandler) ListVessels(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	vessels, total, err := h.svc.ListVessels(c.Request.Context(), page, pageSize)
	if err != nil {
		response.InternalServerError(c.Writer, "failed to list vessels")
		return
	}
	response.SuccessPage(c.Writer, vessels, page, pageSize, total)
}

// ListVesselsByCompany returns vessels owned by a shipping company.
// @Summary      List vessels by company
// @Description  Get paginated vessels for a specific shipping company
// @Tags         Vessel
// @Produce      json
// @Param        shipping_company_id query int true "Shipping company ID"
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(20)
// @Success      200 {object} response.PageResponse{data=[]model.Vessel} "Vessels list"
// @Failure      400 {object} response.Response "Missing company ID"
// @Failure      401 {object} response.Response "Unauthorized"
// @Security     BearerAuth
// @Router       /vessels [get]
func (h *VesselHandler) ListVesselsByCompany(c *gin.Context) {
	companyIDStr := c.Query("shipping_company_id")
	if companyIDStr == "" {
		response.BadRequest(c.Writer, "shipping_company_id is required")
		return
	}
	companyID, err := strconv.ParseInt(companyIDStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid shipping_company_id")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	vessels, total, err := h.svc.ListVesselsByCompany(c.Request.Context(), companyID, page, pageSize)
	if err != nil {
		response.InternalServerError(c.Writer, "failed to list vessels")
		return
	}
	response.SuccessPage(c.Writer, vessels, page, pageSize, total)
}
