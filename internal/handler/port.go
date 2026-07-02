package handler

import (
	"strconv"

	"backend/internal/service"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
)

type PortHandler struct {
	svc service.PortService
}

func NewPortHandler(svc service.PortService) *PortHandler {
	return &PortHandler{svc: svc}
}

// GetPort retrieves a port by its ID.
// @Summary      Get port by ID
// @Description  Return details of a specific port
// @Tags         Port
// @Produce      json
// @Param        id path int true "Port ID"
// @Success      200 {object} response.Response{data=model.Port} "Port found"
// @Failure      400 {object} response.Response "Invalid ID"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      404 {object} response.Response "Port not found"
// @Security     BearerAuth
// @Router       /ports/{id} [get]
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

// ListPorts returns a paginated list of ports.
// @Summary      List ports
// @Description  Get all ports with pagination
// @Tags         Port
// @Produce      json
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(20)
// @Success      200 {object} response.PageResponse{data=[]model.Port} "Ports list"
// @Failure      401 {object} response.Response "Unauthorized"
// @Security     BearerAuth
// @Router       /ports [get]
func (h *PortHandler) ListPorts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 { pageSize = 20 }; if pageSize > 100 { pageSize = 100 }
	ports, total, err := h.svc.ListPorts(c.Request.Context(), page, pageSize)
	if err != nil {
		response.InternalServerError(c.Writer, "failed to list ports")
		return
	}
	response.SuccessPage(c.Writer, ports, page, pageSize, total)
}

// ListPortsByCity returns ports belonging to a specific city.
// @Summary      List ports by city
// @Description  Get paginated ports for a given city
// @Tags         Port
// @Produce      json
// @Param        city_id query int true "City ID"
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(20)
// @Success      200 {object} response.PageResponse{data=[]model.Port} "Ports list"
// @Failure      400 {object} response.Response "Missing city_id"
// @Failure      401 {object} response.Response "Unauthorized"
// @Security     BearerAuth
// @Router       /ports [get]
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
