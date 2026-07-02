package handler

import (
	"strconv"

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

// GetLine retrieves a shipping line by its ID.
// @Summary      Get shipping line by ID
// @Description  Return details of a specific shipping line
// @Tags         Shipping Line
// @Produce      json
// @Param        id path int true "Line ID"
// @Success      200 {object} response.Response{data=model.ShippingLine} "Line found"
// @Failure      400 {object} response.Response "Invalid ID"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      404 {object} response.Response "Line not found"
// @Security     BearerAuth
// @Router       /shipping-lines/{id} [get]
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

// ListLines returns a paginated list of shipping lines.
// @Summary      List shipping lines
// @Description  Get all shipping lines with pagination
// @Tags         Shipping Line
// @Produce      json
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(20)
// @Success      200 {object} response.PageResponse{data=[]model.ShippingLine} "Lines list"
// @Failure      401 {object} response.Response "Unauthorized"
// @Security     BearerAuth
// @Router       /shipping-lines [get]
func (h *ShippingLineHandler) ListLines(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 { pageSize = 20 }; if pageSize > 100 { pageSize = 100 }
	lines, total, err := h.svc.ListLines(c.Request.Context(), page, pageSize)
	if err != nil {
		response.InternalServerError(c.Writer, "failed to list lines")
		return
	}
	response.SuccessPage(c.Writer, lines, page, pageSize, total)
}

// GetPortSequence returns the port sequence JSON of a shipping line.
// @Summary      Get port sequence
// @Description  Retrieve the ordered list of port IDs for a shipping line
// @Tags         Shipping Line
// @Produce      json
// @Param        id path int true "Line ID"
// @Success      200 {object} response.Response{data=object{port_sequence=[]int}} "Port sequence"
// @Failure      400 {object} response.Response "Invalid ID"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      404 {object} response.Response "Line not found or no port sequence"
// @Security     BearerAuth
// @Router       /shipping-lines/{id}/port-sequence [get]
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
