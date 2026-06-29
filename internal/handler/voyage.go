package handler

import (
	"strconv"

	"backend/internal/service"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
)

type VoyageHandler struct {
	svc service.VoyageService
}

func NewVoyageHandler(svc service.VoyageService) *VoyageHandler {
	return &VoyageHandler{svc: svc}
}

// Recommend returns available voyages sorted by remaining capacity.
// @Summary      Recommend voyages
// @Description  Get voyages that can transport required tonnage from start port to end port, sorted by remaining capacity descending
// @Tags         Voyage
// @Produce      json
// @Param        start_port_id query int true "Start port ID"
// @Param        end_port_id query int true "End port ID"
// @Param        required_ton query number true "Required tonnage"
// @Success      200 {object} response.Response{data=[]service.VoyageRecommendation} "List of recommended voyages"
// @Failure      400 {object} response.Response "Missing or invalid parameters"
// @Failure      401 {object} response.Response "Unauthorized"
// @Security     BearerAuth
// @Router       /voyages/recommend [get]
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
