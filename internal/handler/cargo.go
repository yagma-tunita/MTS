package handler

import (
	"strconv"

	"backend/internal/service"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
)

type CargoHandler struct { // 货物管理处理器
	svc service.CargoService
}

func NewCargoHandler(svc service.CargoService) *CargoHandler { // 创建货物处理器
	return &CargoHandler{svc: svc}
}

func (h *CargoHandler) ListAllCargos(c *gin.Context) { // 管理员分页查询所有货物运输记录
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 { pageSize = 20 }; if pageSize > 100 { pageSize = 100 }

	cargos, total, err := h.svc.ListAllCargos(c.Request.Context(), page, pageSize)
	if err != nil {
		response.InternalServerError(c.Writer, "failed to list cargos")
		return
	}
	response.SuccessPage(c.Writer, cargos, page, pageSize, total)
}
