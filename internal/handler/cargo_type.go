package handler

import (
	"strconv"

	"backend/internal/model"
	"backend/internal/service"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
)

type CargoTypeHandler struct {
	svc service.CargoTypeService
}

func NewCargoTypeHandler(svc service.CargoTypeService) *CargoTypeHandler {
	return &CargoTypeHandler{svc: svc}
}

func (h *CargoTypeHandler) ListAll(c *gin.Context) {
	types, err := h.svc.ListAll(c.Request.Context())
	if err != nil {
		response.InternalServerError(c.Writer, "failed to list cargo types")
		return
	}
	response.Success(c.Writer, types)
}

func (h *CargoTypeHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 { pageSize = 20 }; if pageSize > 100 { pageSize = 100 }
	keyword := c.Query("keyword")

	types, total, err := h.svc.List(c.Request.Context(), page, pageSize, keyword)
	if err != nil {
		response.InternalServerError(c.Writer, "failed to list cargo types")
		return
	}
	response.SuccessPage(c.Writer, types, page, pageSize, total)
}

func (h *CargoTypeHandler) Create(c *gin.Context) {
	var t model.CargoType
	if err := c.ShouldBindJSON(&t); err != nil {
		response.BadRequest(c.Writer, "invalid request body")
		return
	}
	if err := h.svc.Create(c.Request.Context(), &t); err != nil {
		response.InternalServerError(c.Writer, "failed to create cargo type")
		return
	}
	response.Success(c.Writer, t)
}

func (h *CargoTypeHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid id")
		return
	}
	var t model.CargoType
	if err := c.ShouldBindJSON(&t); err != nil {
		response.BadRequest(c.Writer, "invalid request body")
		return
	}
	t.TypeID = id
	if err := h.svc.Update(c.Request.Context(), &t); err != nil {
		response.InternalServerError(c.Writer, "failed to update cargo type")
		return
	}
	response.Success(c.Writer, t)
}

func (h *CargoTypeHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid id")
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.InternalServerError(c.Writer, "failed to delete cargo type")
		return
	}
	response.Success(c.Writer, gin.H{"message": "cargo type deleted"})
}
