package handler

import (
	"strconv"

	"backend/internal/model"
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

	keyword := c.Query("keyword")
	cargos, total, err := h.svc.ListAllCargos(c.Request.Context(), page, pageSize, keyword)
	if err != nil {
		response.InternalServerError(c.Writer, "failed to list cargos")
		return
	}
	response.SuccessPage(c.Writer, cargos, page, pageSize, total)
}

func (h *CargoHandler) CreateCargo(c *gin.Context) {
	var cargo model.OrderCargo
	if err := c.ShouldBindJSON(&cargo); err != nil {
		response.BadRequest(c.Writer, "invalid request body")
		return
	}
	if err := h.svc.CreateCargo(c.Request.Context(), &cargo); err != nil {
		response.InternalServerError(c.Writer, "failed to create cargo")
		return
	}
	response.Success(c.Writer, cargo)
}

func (h *CargoHandler) DeleteCargo(c *gin.Context) { // 管理员删除货物
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid cargo id")
		return
	}
	if err := h.svc.DeleteCargo(c.Request.Context(), id); err != nil {
		response.InternalServerError(c.Writer, "failed to delete cargo")
		return
	}
	response.Success(c.Writer, gin.H{"message": "cargo deleted"})
}
