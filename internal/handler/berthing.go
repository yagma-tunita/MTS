package handler

import (
	"strconv"
	"time"

	"backend/internal/service"
	"backend/pkg/errors"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
)

type BerthingHandler struct { // 靠泊记录处理器
	svc service.VoyageBerthingService
}

func NewBerthingHandler(svc service.VoyageBerthingService) *BerthingHandler { // 创建靠泊处理器
	return &BerthingHandler{svc: svc}
}

type updateActualTimesRequest struct { // 更新实际时间请求体
	ActualArrivalTime   *time.Time `json:"actual_arrival_time"`
	ActualDepartureTime *time.Time `json:"actual_departure_time"`
}

func (h *BerthingHandler) ListByCompany(c *gin.Context) {
	role, _ := c.Get("role")
	if role != "shipping" {
		response.ErrorWithCode(c.Writer, 1002, "only shipping companies can access")
		return
	}
	userID, _ := c.Get("user_id")
	companyID, ok := userID.(int64)
	if !ok {
		response.ErrorWithCode(c.Writer, 1001, "invalid user identity")
		return
	}
	berthings, err := h.svc.ListByShippingCompany(c.Request.Context(), companyID)
	if err != nil {
		response.InternalServerError(c.Writer, "failed to list berthings")
		return
	}
	response.Success(c.Writer, berthings)
}

func (h *BerthingHandler) UpdateActualTimes(c *gin.Context) { // 更新靠泊记录的实际到达/出发时间
	role, _ := c.Get("role")
	if role != "shipping" {
		response.ErrorWithCode(c.Writer, errors.CodeForbidden, "only shipping companies can update times"); return
	}
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid berthing id")
		return
	}
	var req updateActualTimesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c.Writer, "invalid request body")
		return
	}
	if err := h.svc.UpdateActualTimes(c.Request.Context(), id, req.ActualArrivalTime, req.ActualDepartureTime); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.ErrorWithCode(c.Writer, appErr.Code, appErr.Message)
			return
		}
		response.InternalServerError(c.Writer, "failed to update actual times")
		return
	}
	response.Success(c.Writer, gin.H{"message": "actual times updated"})
}
